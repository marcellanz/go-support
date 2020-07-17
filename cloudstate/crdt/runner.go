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
	stream        protocol.Crdt_HandleServer
	context       *Context
	stateReceived bool
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
	r.stateReceived = true
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
func (r *runner) handleCancellation(cancelled *protocol.StreamCancelled) error {
	id := CommandId(cancelled.GetId())
	ctx := r.context.streamedCtx[id]
	delete(r.context.streamedCtx, id)
	if ctx.cancel == nil {
		return r.sendCancelledMessage(&protocol.CrdtStreamCancelledResponse{
			CommandId: id.Value(),
		})
	}
	if err := ctx.cancelled(); err != nil {
		return err
	}
	ctx.deactivate()
	err := r.sendCancelledMessage(&protocol.CrdtStreamCancelledResponse{
		CommandId:   id.Value(),
		StateAction: ctx.stateAction(),
		SideEffects: ctx.sideEffects,
	})
	if err != nil {
		return err
	}
	ctx.clearSideEffect()
	return nil
}

func (r *runner) handleCommand(cmd *protocol.Command) (streamError error) {
	ctx := r.context.commandContextFor(cmd)
	reply, err := ctx.runCommand(cmd)
	// on any error, the context gets deactivated
	if err != nil {
		ctx.deactivate() // this will close the stream
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
		return r.sendStreamedMessage(&protocol.CrdtStreamedMessage{
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
	// after the command handling, and any stateActions should get handled by existing change handlers
	// => that is the reason the scala impl copies them over later. TODO: explain in SPEC feedback
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
			msg := &protocol.CrdtStreamedMessage{
				CommandId:    ctx.CommandId.Value(),
				ClientAction: clientAction,
			}
			if err := r.sendStreamedMessage(msg); err != nil {
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
