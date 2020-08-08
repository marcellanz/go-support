//
// Copyright 2020 Lightbend Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package crdt

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
)

var ErrCtxFailCalled = errors.New("context failed")
var ErrStateChanged = errors.New("CRDT change not allowed")

/**
 * register an on change callback for this command.
 *
 * <p>The callback will be invoked any time the CRDT changes. The callback may inspect the CRDT,
 * but any attempt to modify the CRDT will be ignored and the CRDT will crash.
 *
 * <p>If the callback returns a value, that value will be sent down the stream. Alternatively, the
 * callback may forward messages to other entities via the passed in {@link SubscriptionContext}.
 * The callback may also emit side effects to other entities via that context.
 *
 * @param subscriber The subscriber callback.
 */
type ChangeFunc func(c *CommandContext) (*any.Any, error)
type CancelFunc func(c *CommandContext) error

type CommandContext struct {
	*Context
	CommandId CommandId
	change    ChangeFunc
	cancel    CancelFunc
	// ended means, we will send a streamed message
	// where we mark the message as the last one in the stream
	// and therefore, the streamed command has ended.
	ended       bool
	cmd         *protocol.Command
	forward     *protocol.Forward
	sideEffects []*protocol.SideEffect
}

func (c *CommandContext) runCommand(cmd *protocol.Command) (*any.Any, error) {
	// unmarshal the commands message
	msgName := strings.TrimPrefix(cmd.GetPayload().GetTypeUrl(), "type.googleapis.com"+"/")
	messageType := proto.MessageType(msgName)
	message, ok := reflect.New(messageType.Elem()).Interface().(proto.Message)
	if !ok {
		return nil, fmt.Errorf("messageType is no proto.Message: %v", messageType)
	}
	err := proto.Unmarshal(cmd.Payload.Value, message)
	if err != nil {
		return nil, err
	}
	return c.Instance.HandleCommand(c, cmd.Name, message)
}

// ChangeFunc sets the function to be called whenever the CRDT is changed.
// For non-streamed contexts this is a no operation.
func (c *CommandContext) ChangeFunc(f ChangeFunc) {
	if !c.Streamed() {
		return
	}
	c.change = f
}

func (c *CommandContext) Command() *protocol.Command {
	return c.cmd
}

func (c *CommandContext) Streamed() bool {
	if c.cmd == nil {
		return false
	}
	return c.cmd.Streamed
}

// CancelFunc registers an on cancel handler for this command.
//
// The registered function will be invoked if the client initiates a stream cancel. It will not
// be invoked if the entity cancels the stream itself via {@link SubscriptionContext#endStream()} from an {@link
// StreamedCommandContext#onChange(Function)} callback.
// <p>An on cancel callback may update the CRDT, and may emit side effects via the passed in
// {@link StreamCancelledContext}.
// @param effect The effect to perform when this stream is cancelled.
func (c *CommandContext) CancelFunc(f CancelFunc) {
	if !c.Streamed() {
		return
	}
	c.cancel = f
}

func (c *CommandContext) EndStream() {
	if !c.Streamed() {
		return
	}
	c.ended = true
}

func (c *CommandContext) Forward(f *protocol.Forward) {
	if c.forward != nil {
		c.fail(errors.New("this context has already forwarded"))
	}
	c.forward = f
}

func (c *CommandContext) SideEffect(e *protocol.SideEffect) {
	c.sideEffects = append(c.sideEffects, e)
}

func (c *CommandContext) clearSideEffect() {
	c.sideEffects = make([]*protocol.SideEffect, 0, cap(c.sideEffects)) // TODO: should we decrease that?
}

func (c *CommandContext) changed() (reply *any.Any, err error) {
	// spec impl: checkActive()
	reply, err = c.change(c)
	if c.crdt.HasDelta() {
		// the user is not allowed to change the CRDT.s
		err = ErrStateChanged
	}
	return
}

/**
 * TODO: rewrite as Go documentation
 * register an on cancel callback for this command.
 *
 * <p>This will be invoked if the client initiates a stream cancel. It will not be invoked if the
 * entity cancels the stream itself via {@link SubscriptionContext#endStream()} from an {@link
 * StreamedCommandContext#onChange(Function)} callback.
 *
 * <p>An on cancel callback may update the CRDT, and may emit side effects via the passed in
 * {@link StreamCancelledContext}.
 *
 * @param effect The effect to perform when this stream is cancelled.
 */
func (c *CommandContext) cancelled() error {
	// spec impl: checkActive()
	return c.cancel(c)
}

func (c *Context) commandContextFor(cmd *protocol.Command) *CommandContext {
	return &CommandContext{
		Context:     c,
		cmd:         cmd,
		CommandId:   CommandId(cmd.Id),
		sideEffects: make([]*protocol.SideEffect, 0),
	}
}

func (c *CommandContext) trackChanges() {
	c.streamedCtx[c.CommandId] = c
}

func (c *CommandContext) clientActionFor(reply *any.Any) (*protocol.ClientAction, error) {
	if c.failed != nil {
		return &protocol.ClientAction{
			Action: &protocol.ClientAction_Failure{
				Failure: &protocol.Failure{
					CommandId:   c.CommandId.Value(),
					Description: c.failed.Error(),
				},
			},
		}, nil
	}
	if reply != nil {
		if c.forward != nil {
			// spec impl: "Both a reply was returned, and a forward message was sent, choose one or the other."
			// TODO notallowed: "This context has already forwarded."
			return nil, errors.New("this context has already forwarded")
		}
		return &protocol.ClientAction{
			Action: &protocol.ClientAction_Reply{
				Reply: &protocol.Reply{
					Payload: reply,
				},
			},
		}, nil
	}
	if c.forward != nil {
		return &protocol.ClientAction{
			Action: &protocol.ClientAction_Forward{
				Forward: c.forward,
			},
		}, nil
	}
	return nil, nil
}

func (c *CommandContext) stateAction() *protocol.CrdtStateAction {
	if c.crdt == nil {
		return nil
	}
	if c.created && c.crdt.HasDelta() {
		c.created = false
		if c.deleted {
			c.crdt = nil
			return nil
		}
		c.crdt.resetDelta()
		return &protocol.CrdtStateAction{
			Action: &protocol.CrdtStateAction_Create{
				Create: c.crdt.State(),
			},
		}
	}
	if c.created && c.deleted {
		c.created = false
		c.crdt = nil
		return nil
	}
	if c.deleted {
		return &protocol.CrdtStateAction{
			Action: &protocol.CrdtStateAction_Delete{Delete: &protocol.CrdtDelete{}},
		}
	}
	if c.crdt.HasDelta() {
		delta := c.crdt.Delta()
		c.crdt.resetDelta()
		return &protocol.CrdtStateAction{
			Action: &protocol.CrdtStateAction_Update{Update: delta},
		}
	}
	return nil
}
