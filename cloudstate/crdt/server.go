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

func (r *runner) cancelledMessage(msg *protocol.CrdtStreamCancelledResponse) error {
	return r.stream.Send(&protocol.CrdtStreamOut{
		Message: &protocol.CrdtStreamOut_StreamCancelledResponse{
			StreamCancelledResponse: msg,
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

			if err := runner.handleCommand(m.Command); err != nil {
				return err
			}
			// after the command handling, and any clientActions
			// should get handled by existing change handlers
			// => that is the reason the scala impl copies them over later. TODO: explain in SPEC feedback
			if err := runner.handleChange(); err != nil {
				return err
			}
		case *protocol.CrdtStreamIn_StreamCancelled:
			if err := runner.handleCancellation(m.StreamCancelled); err != nil {
				return err
			}
		case *protocol.CrdtStreamIn_Init:
			return errors.New("duplicate init message for the same entity")
		case nil:
			return errors.New("empty message received")
		default:
			return fmt.Errorf("unknown message received: %v", msg.GetMessage())
		}
	}
}

func (r *runner) handleChange() error {
	for _, streamCtx := range r.context.streamedCtx {
		if streamCtx.change == nil {
			continue
		}
		// get reply from stream contexts ChangeFuncs
		reply, err := streamCtx.changed()
		if err != nil {
			streamCtx.deactivate() // TODO: why/how?
		}
		if errors.Is(err, ErrFailCalled) {
			reply = nil
		} else if err != nil {
			return err
		}

		clientAction := streamCtx.clientActionFor(reply)
		if streamCtx.failed != nil {
			// remove cancel and change funcs: TODO: should we remove the stream context?
			streamCtx.change = nil
			streamCtx.cancel = nil
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
				// ended means, we will send a streamed message
				// where we mark the message as the last one in the stream
				// and therefore, the streamed command has ended.
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
			streamCtx.clearSideEffect()
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
		created:     false,
		ctx:         r.stream.Context(), // this context is stable as long as the runner runs
		active:      true,
		streamedCtx: make(map[CommandId]*StreamContext),
	}
	// the CrdtInit msg may have an initial state
	if state := init.GetState(); state != nil {
		if err := r.handleState(state); err != nil {
			return err
		}
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
	id := CommandId(cancelled.GetId())
	ctx := r.context.streamedCtx[id]
	delete(r.context.streamedCtx, id)
	if ctx.cancel == nil {
		return r.cancelledMessage(&protocol.CrdtStreamCancelledResponse{
			CommandId: id.Value(),
		})
	}

	if err := ctx.cancelled(); err != nil {
		return err
	}
	ctx.deactivate()
	stateAction := ctx.stateAction()
	err := r.cancelledMessage(&protocol.CrdtStreamCancelledResponse{
		CommandId:   id.Value(),
		StateAction: stateAction,
		SideEffects: ctx.sideEffects,
	})
	if err != nil {
		return err
	}
	ctx.clearSideEffect()
	return r.handleChange()
}

func (r *runner) handleCommand(cmd *protocol.Command) error {
	id := CommandId(cmd.GetId())
	if cmd.GetStreamed() {
		r.context.enableStreamFor(id)
	}
	// TODO: goon

	return nil
}
