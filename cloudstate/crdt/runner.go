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

	"github.com/cloudstateio/go-support/cloudstate/protocol"
)

type runner struct {
	stream  protocol.Crdt_HandleServer
	context *Context
}

// A CrdtState message is sent to indicate the user function should replace its current value with this value. If the user function
// does not have a current value, either because the commandContextFor function didn't send one and the user function hasn't
// updated the value itself in response to a command, or because the value was deleted, this must be sent before
// any deltas.
func (r *runner) handleState(state *protocol.CrdtState) error {
	if r.context.crdt == nil {
		r.context.crdt = newFor(state)
	}
	// state has to match the state type, applyState will err if not.
	if err := r.context.crdt.applyState(state); err != nil {
		return err
	}
	r.context.Instance.Set(r.context, r.context.crdt)
	return nil
}

// A delta to be applied to the current value. It may be sent at any time as long as the user function already has
// value.
func (r *runner) handleDelta(delta *protocol.CrdtDelta) error {
	if r.context.crdt == nil {
		return fmt.Errorf("received delta for crdt before it was created: %v", r.context.Entity)
	}
	return r.context.crdt.applyDelta(delta)
}

// A stream has been cancelled.
// A command handler may also register an onCancel callback to be notified when the stream
// is cancelled. The cancellation callback handler may update the crdt. This is useful if the crdt
// is being used to track connections, for example, when using Vote CRDTs to track a users online status.

// handleCancellation
func (r *runner) handleCancellation(cancelled *protocol.StreamCancelled) error {
	id := CommandId(cancelled.GetId())
	ctx := r.context.streamedCtx[id]
	// the cancelled stream is not allowed to handle changes, so we remove it.
	delete(r.context.streamedCtx, id)
	if ctx.cancel == nil {
		return r.sendCancelledMessage(&protocol.CrdtStreamCancelledResponse{
			CommandId: id.Value(),
		})
	}
	// notify the user about the cancellation.
	if err := ctx.cancelled(); err != nil {
		return err
	}
	ctx.deactivate()
	stateAction := ctx.stateAction()
	err := r.sendCancelledMessage(&protocol.CrdtStreamCancelledResponse{
		CommandId:   id.Value(),
		StateAction: stateAction,
		SideEffects: ctx.sideEffects,
	})
	if err != nil {
		return err
	}
	if stateAction != nil {
		return r.handleChange()
	}
	return nil
}

// handleCommand handles the received command.
// Cloudstate CRDTs support handling server streamed calls, that is, when the gRPC service call for a CRDT marks the return type as streamed. When a user function receives a streamed message, it is allowed to update the CRDT, on two occasions - when the call is first received, and when the client cancels the stream. If it wishes to make updates at other times, it can do so by emitting effects with the streamed messages that it sends down the stream.
// A user function can send a message down a stream in response to anything, however the Cloudstate supplied support libraries only allow sending messages in response to the CRDT changing. In this way, use cases that require monitoring the state of a CRDT can be implemented.
func (r *runner) handleCommand(cmd *protocol.Command) (streamError error) {
	if r.context.EntityId != EntityId(cmd.EntityId) {
		return fmt.Errorf("given cmd.EntityId: %s does not match the initialized entityId: %s", cmd.EntityId, r.context.EntityId)
	}
	if _, ok := r.context.streamedCtx[CommandId(cmd.Id)]; ok {
		return fmt.Errorf("the command has already been handled: %d", cmd.Id)
	}
	ctx := r.context.commandContextFor(cmd)
	reply, err := ctx.runCommand(cmd)
	// TODO: error handling has to be clarified.
	// 	It seems, CRDT streams stopped for any error, even client failures.
	//	see: https://github.com/cloudstateio/cloudstate/pull/392
	if err != nil {
		// on any error, the context gets deactivated. TODO: hmm this seems wrong, yes?
		ctx.deactivate() // this will close the stream
		ctx.fail(err)
	}
	// if the user function has failed, a client action failure will be sent
	if ctx.failed != nil {
		reply = nil
	} else if err != nil {
		// on any other error, we return and close the stream
		return err
	}
	clientAction, err := ctx.clientActionFor(reply)
	if err != nil {
		return err
	}
	if ctx.failed != nil {
		return r.sendCrdtReply(&protocol.CrdtReply{
			CommandId:    ctx.CommandId.Value(),
			ClientAction: clientAction, // this is a ClientAction_Failure
		})
	}
	stateAction := ctx.stateAction()
	err = r.sendCrdtReply(&protocol.CrdtReply{
		CommandId:    ctx.CommandId.Value(),
		ClientAction: clientAction,
		SideEffects:  ctx.sideEffects,
		StateAction:  stateAction,
		Streamed:     ctx.Streamed(),
	})
	if err != nil {
		return err
	}
	ctx.clearSideEffect()
	// after the command handling, and any stateActions should get handled by existing change handlers
	// => that is the reason the scala impl copies them over later. TODO: explain in SPEC feedback
	//
	// so this makes sense, as we don't support client streaming RPC, so a command is handled
	// once and once only.
	if stateAction != nil {
		if err := r.handleChange(); err != nil {
			return err
		}
	}
	if ctx.Streamed() {
		ctx.trackChanges()
	}
	return nil
}

func (r *runner) handleChange() error {
	for _, ctx := range r.context.streamedCtx {
		if ctx.change == nil {
			continue
		}
		reply, err := ctx.changed()
		if err != nil {
			ctx.deactivate() // TODO: why/how? failed, active, deleted; hows that to work?
		}
		if errors.Is(err, ErrCtxFailCalled) {
			reply = nil
		} else if err != nil {
			return err
		}
		clientAction, err := ctx.clientActionFor(reply)
		if err != nil {
			return err
		}
		if ctx.failed != nil {
			delete(ctx.streamedCtx, ctx.CommandId)
			if err := r.sendStreamedMessage(&protocol.CrdtStreamedMessage{
				CommandId:    ctx.CommandId.Value(),
				ClientAction: clientAction,
			}); err != nil {
				return err
			}
			continue
		}
		if clientAction != nil || ctx.ended || len(ctx.sideEffects) > 0 {
			if ctx.ended {
				delete(ctx.streamedCtx, ctx.CommandId)
			}
			msg := &protocol.CrdtStreamedMessage{
				CommandId:    ctx.CommandId.Value(),
				ClientAction: clientAction,
				SideEffects:  ctx.sideEffects,
				EndStream:    ctx.ended,
			}
			if err := r.sendStreamedMessage(msg); err != nil {
				return err
			}
			ctx.clearSideEffect()
			continue
		}
	}
	return nil
}

func (r *runner) sendStreamedMessage(msg *protocol.CrdtStreamedMessage) error {
	return r.stream.Send(&protocol.CrdtStreamOut{
		Message: &protocol.CrdtStreamOut_StreamedMessage{
			StreamedMessage: msg,
		},
	})
}

func (r *runner) sendCancelledMessage(msg *protocol.CrdtStreamCancelledResponse) error {
	return r.stream.Send(&protocol.CrdtStreamOut{
		Message: &protocol.CrdtStreamOut_StreamCancelledResponse{
			StreamCancelledResponse: msg,
		},
	})
}

func (r *runner) sendCrdtReply(reply *protocol.CrdtReply) error {
	return r.stream.Send(&protocol.CrdtStreamOut{
		Message: &protocol.CrdtStreamOut_Reply{
			Reply: reply,
		},
	})
}
