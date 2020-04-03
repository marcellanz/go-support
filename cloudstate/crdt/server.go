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
	"io"
	"sync"

	"github.com/cloudstateio/go-support/cloudstate/protocol"
)

type Server struct {
	// entities has descriptions of entities registered by service names
	entities map[ServiceName]*Entity
	mu       sync.RWMutex
}

func NewServer() *Server {
	return &Server{
		entities: make(map[ServiceName]*Entity),
	}
}

// CrdtEntities can be registered to a server that handles crdt entities by a ServiceName.
// Whenever a internalCRDT.Server receives an CrdInit for an instance of a crdt entity identified by its
// EntityId and a ServiceName, the internalCRDT.Server handles such entities through their lifecycle.
// The handled entities value are captured by a context that is held fo each of them.
func (s *Server) Register(e *Entity, serviceName ServiceName) error {
	if e.EntityFunc == nil {
		return fmt.Errorf("the entity has to define an EntityFunc but did not")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.entities[serviceName]; exists {
		return fmt.Errorf("serviceName already registered: %v", serviceName)
	}
	s.entities[serviceName] = e
	return nil
}

// Entity captures an Entity with its ServiceName.
// It is used to be registered as an crdt entity on a Cloudstate instance.
type Entity struct {
	// ServiceName is the fully qualified name of the service that implements this entities interface.
	// Setting it is mandatory.
	ServiceName ServiceName
	// EntityFunc
	EntityFunc func(id EntityId) interface{}
	// DefaultFunc is a factory method to create the crdt to be used for this entity.
	DefaultFunc func(c *Context) CRDT
}

type runner struct {
	context       *Context
	stream        protocol.Crdt_HandleServer
	stateReceived bool
}

func (r *runner) streamedMessage(msg *protocol.CrdtStreamedMessage) error {
	return r.stream.Send(&protocol.CrdtStreamOut{
		Message: &protocol.CrdtStreamOut_StreamedMessage{
			StreamedMessage: msg,
		},
	})
}

// After invoking handle, the first message sent will always be a CrdtInit message, containing the entity ID, and,
// if it exists or is available, the current value of the entity. After that, one or more commands may be sent,
// as well as deltas as they arrive, and the entire value if either the entity is created, or the proxy wishes the
// user function to replace its entire value.
// The user function must respond with one reply per command in. They do not necessarily have to be sent in the same
// order that the commands were sent, the command ID is used to correlate commands to replies.
func (s *Server) Handle(stream protocol.Crdt_HandleServer) (handleErr error) {
	defer func() {
		if r := recover(); r != nil {
			handleErr = sendFailureAndReturnWith(fmt.Errorf("CrdtServer.Handle panic-ked with: %v", r), stream)
		}
	}()
	if err := s.handle(stream); err != nil {
		return sendFailureAndReturnWith(err, stream)
	}
	return nil
}

func (s *Server) handle(stream protocol.Crdt_HandleServer) error {
	first, err := stream.Recv()
	if err == io.EOF { // stream has ended
		return nil
	}
	if err != nil {
		return err
	}
	runner := &runner{stream: stream}

	// first, always a CrdtInit message must be received.
	switch m := first.GetMessage().(type) {
	case *protocol.CrdtStreamIn_Init:
		if err = s.handleInit(m.Init, runner); err != nil {
			return fmt.Errorf("handling of CrdtInit failed with: %w", err)
		}
	default:
		return fmt.Errorf("a message was received without having an CrdtInit message first: %v", m)
	}
	if runner.context.EntityId == "" { // never happens, but if.
		return fmt.Errorf("a message was received without having a CrdtInit message handled before: %v", first.GetMessage())
	}

	// handle all other messages after a CrdtInit message has been received.
	ctx := runner.stream.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		// TODO: what is all the active, ended, failed mean? => we close the stream
		if runner.context.deleted || !runner.context.active {
			return nil
		}
		if runner.context.failed != nil {
			// failed means deactivated, so we never get this far
			return nil
		}
		msg, err := runner.stream.Recv()
		if err == io.EOF { // stream has ended
			return nil
		}
		if err != nil {
			return err
		}

		// TODO: how to handle an inactive entity?
		switch m := msg.GetMessage().(type) {
		case *protocol.CrdtStreamIn_State:
			if err := runner.handleState(m.State); err != nil {
				return err
			}
			if err := runner.handleChange(); err != nil {
				return err
			}
		case *protocol.CrdtStreamIn_Changed:
			if !runner.stateReceived {
				return errors.New("received a CrdtDelta message without having a CrdtState ever received")
			}
			if err := runner.handleDelta(m.Changed); err != nil {
				return err
			}
			if err := runner.handleChange(); err != nil {
				return err
			}
		case *protocol.CrdtStreamIn_Deleted:
			// Delete the entity. May be sent at any time. The user function should clear its value when it receives this.
			// A proxy may decide to terminate the stream after sending this.
			runner.context.delete()
		case *protocol.CrdtStreamIn_Command:
			// A command, may be sent at any time.
			id := CommandId(m.Command.GetId())
			if m.Command.GetStreamed() {
				runner.context.enableStreamFor(id)
			}

			// TODO: handle command

			if err := runner.handleChange(); err != nil {
				return err
			}

			// CRDTs may only be updated in command handlers and stream cancellation callbacks.

			// so there are entities that have a entity runner
			// and at the same time multiple commands running aka. streamedCtx
			// and an entity instance might decide to subscribe running commands
			// separately
			//
			// | command-foo-1 |
			//		\-> subscribe(one_way)
			// | command-foo-2 |
			//		\-> doesn't
			//	| command-bar-1 |
			//		\-> subscribe(another_way)

			// distinguish between streamedCtx and non-streamedCtx commands

			// - handle command
			// - in any error, deactivate the running context

			// - create ClientAction depending on the reply
			// 	- error produces an Action.Failure
			// - on no error
			//  - a reply and a forward can't be defined at the same time => fail
			//  - otherwise a ClientAction.Action.Reply is sent
			//  - if a forward is defined, send it as a ClientAction.Action.Forward
			// 	- otherwise fail with that no reply or forward was defined

			// a command can result in a protocol.CrdtStateAction_Create

			// if the ctx has an error
			// - send a CrdtStreamOut.Message.Reply with commandId and clientAction
			// 	 which would be a ClientAction.Action.Failure
			// else
			// - createCrdtAction
			// - collect streamedMessages by notifySubscribers
			//

			// if failed: reply
			// - verifyNoDelta

			// on any error not handled
			// =>  CrdtStreamOut(CrdtStreamOut.Message.Failure(Failure(description = err.getMessage)))

			// it seems this can happen even before any stream was set

		case *protocol.CrdtStreamIn_StreamCancelled:
			// a Cancellation can result in a protocol.CrdtStateAction_Create

			// calls cancelListeners
			// creates a createCrdtAction
			//   and creates a CrdtStreamOut.Message.StreamCancelledResponse with
			//		action, command-id, and effects
			//   and maps returns of subscribers to CrdtStreamOut.Message.StreamedMessage s
			// otherwise
			//  CrdtStreamOut.Message.StreamCancelledResponse with
			//		command-id, and effects
			// on no cancellers
			// 	rends CrdtStreamCancelledResponse(cancelled.id)

			if err := runner.handleCancellation(m.StreamCancelled); err != nil {
				return err
			}
		case *protocol.CrdtStreamIn_Init:
			return errors.New("duplicate enableStreamFor event for the same entity")
		case nil:
			return errors.New("empty message received")
		default:
			return errors.New("unknown message received")
		}
	}
}

// notify
// Streamed command handlers

// for any running streamedCtx commands, their change function get called
// and return any clientAction they might send in return to the stream change
// - a reply
// - a forward
// - nothing

// onChange lets onChange listeners return a value (any.Any)
// that gets "sent down the s"

// onChange is done by streaming commands only, so a non-streamedCtx
// command might return a same kind of value, but on a streamedCtx
// context, onChange listeners can send them too.

// produces: CrdtStreamOut.Message.StreamedMessage

// notifySubscribers
// - callback and get a reply
// - get a clientAction which is a Reply or a Forward but not both
// - if an error exists, form a failure reply
//  - remove subscribers and cancelListeners
// 	- with an error, send the error and the command id
// - a client action and/or side Effects are present, send them as a streamedCtx message
// - if the context is ended, remove subscribers and cancelListeners

// -
// -

//_ = &protocol.CrdtStreamOut{
//	Message: &protocol.CrdtStreamOut_StreamedMessage{
//		StreamedMessage: &protocol.CrdtStreamedMessage{
//			CommandId:    0,
//			ClientAction: nil,
//			SideEffects:  nil,
//			EndStream:    false,
//		},
//	},
//}
func (r *runner) handleChange() error {
	// even if a runner.context is not a streaming context, we want to notify others
	for _, streamCtx := range r.context.streamedCtx { // streamedCtx command context
		// get reply from stream contexts ChangeFuncs
		reply, err := streamCtx.changed()
		if err != nil {
			streamCtx.deactivate() // TODO: why/how?
		}
		if errors.Is(err, ErrFailedCalled) {
			reply = nil
		} else if err != nil {
			return err
		}
		// clientAction
		var clientAction *protocol.ClientAction
		if streamCtx.failed != nil {
			clientAction = &protocol.ClientAction{
				Action: &protocol.ClientAction_Failure{
					Failure: &protocol.Failure{
						CommandId:   streamCtx.CommandId.Value(),
						Description: streamCtx.failed.Error(),
					},
				},
			}
		} else {
			if reply != nil {
				if streamCtx.forward != nil {
					// spec impl: "Both a reply was returned, and a forward message was sent, choose one or the other."
					// TODO notallowed: "This context has already forwarded."
				} else {
					clientAction = &protocol.ClientAction{
						Action: &protocol.ClientAction_Reply{
							Reply: &protocol.Reply{
								Payload: reply,
							},
						},
					}
				}
			} else if streamCtx.forward != nil {
				clientAction = &protocol.ClientAction{
					Action: &protocol.ClientAction_Forward{
						Forward: streamCtx.forward,
					},
				}
			}
		}
		// end of createClientAction
		if streamCtx.failed != nil {
			// remove cancel and change funcs
			msg := &protocol.CrdtStreamedMessage{
				CommandId:    streamCtx.CommandId.Value(),
				ClientAction: clientAction,
			}
			if err := r.streamedMessage(msg); err != nil {
				return err
			}
			continue
		}
		if clientAction != nil || streamCtx.ended || len(streamCtx.sideEffects) > 0 {
			if streamCtx.ended {
				// ended means, we will send (below) a streamed message
				// where we mark the message as the last one in the stream
				// and therefore, the streamed command has ended
				delete(streamCtx.streamedCtx, streamCtx.CommandId) // TODO: do we race here somehow?
			}
			msg := &protocol.CrdtStreamedMessage{
				CommandId:    streamCtx.CommandId.Value(),
				ClientAction: clientAction,
				SideEffects:  streamCtx.sideEffects,
				EndStream:    streamCtx.ended,
			}
			if err := r.streamedMessage(msg); err != nil {
				return err
			}
			continue
		}
	}
	return nil
}

func (s *Server) handleInit(init *protocol.CrdtInit, r *runner) error {
	serviceName := ServiceName(init.GetServiceName())
	s.mu.RLock()
	entity, exists := s.entities[serviceName]
	s.mu.RUnlock()
	if !exists {
		return fmt.Errorf("received a command for an unknown crdt service: %v", serviceName)
	}
	if entity.EntityFunc == nil {
		return fmt.Errorf("entity.EntityFunc not defined: %v", serviceName)
	}
	id := EntityId(init.GetEntityId())
	r.context = &Context{
		EntityId:    id,
		Entity:      entity,
		Instance:    entity.EntityFunc(id),
		origin:      Undefined,
		ctx:         r.stream.Context(), // this context is stable as long as the runner runs
		active:      true,
		streamedCtx: make(map[CommandId]*StreamContext),
	}
	// the CrdtInit msg may have an initial state
	if state := init.GetState(); state != nil {
		if err := r.handleState(state); err != nil {
			return err
		}
		r.context.origin = Proxy
	}
	// the user entity can provide a crdt through a default function if none is set.
	r.context.initDefault()
	return nil
}

// A CrdtState message is sent to indicate the user function should replace its current value with this value. If the user function
// does not have a current value, either because the enableStreamFor function didn't send one and the user function hasn't
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
// A streamedCtx command handler may also register an onCancel callback to be notified when the stream
// is cancelled. The cancellation callback handler may update the crdt. This is useful if the crdt
// is being used to track connections, for example, when using Vote CRDTs to track a users online status.
func (r *runner) handleCancellation(cancelled *protocol.StreamCancelled) error {

	cancelled.GetId()

	return nil
}
