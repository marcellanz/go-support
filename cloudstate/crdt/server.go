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
	"github.com/golang/protobuf/ptypes/any"
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
	// EntityFunc creates a new entity.
	EntityFunc func(id EntityId) interface{}
	// DefaultFunc is a factory method to create the crdt to be used for this entity.
	DefaultFunc func(c *Context) CRDT
	CommandFunc func(entity interface{}, ctx *CommandContext, name string, msg interface{}) (*any.Any, error)
}

type runner struct {
	stream        protocol.Crdt_HandleServer
	context       *Context
	stateReceived bool
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
	if err == io.EOF { // the stream has ended
		return nil
	}
	if err != nil {
		return err
	}

	runner := &runner{stream: stream}
	switch m := first.GetMessage().(type) {
	case *protocol.CrdtStreamIn_Init:
		// first, always a CrdtInit message must be received.
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
	for {
		select {
		case <-runner.stream.Context().Done():
			return runner.stream.Context().Err()
		default:
		}
		if runner.context.deleted || !runner.context.active {
			return nil
		}
		if runner.context.failed != nil {
			// failed means deactivated. we may never get this far.
			return nil
		}
		msg, err := runner.stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
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
			runner.context.Delete()
		case *protocol.CrdtStreamIn_Command:
			// A command, may be sent at any time.
			// The CRDT is allowed to be changed.
			if err := runner.handleCommand(m.Command); err != nil {
				return err
			}
		case *protocol.CrdtStreamIn_StreamCancelled:
			// The CRDT is allowed to be changed.
			if err := runner.handleCancellation(m.StreamCancelled); err != nil {
				return err
			}
			if err := runner.handleChange(); err != nil {
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
		streamedCtx: make(map[CommandId]*CommandContext),
	}
	// the init msg may have an initial state
	if state := init.GetState(); state != nil {
		if err := r.handleState(state); err != nil {
			return err
		}
	}
	// the user entity can provide a CRDT through a default function if none is set.
	r.context.initDefault()
	return nil
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

func (r *runner) handleCommand(cmd *protocol.Command) error {
	ctx := r.context.commandContextFor(cmd)
	reply, err := ctx.runCommand(cmd)
	if err != nil {
		ctx.deactivate()
	}
	if errors.Is(err, ErrFailCalled) {
		reply = nil // this triggers a ClientAction_Failure be created later by clientActionFor
	} else if err != nil {
		return err
	}
	clientAction := ctx.clientActionFor(reply)
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
		err := r.handleChange()
		if err != nil {
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
			ctx.deactivate() // TODO: why/how?
		}
		if errors.Is(err, ErrFailCalled) {
			reply = nil
		} else if err != nil {
			return err
		}
		clientAction := ctx.clientActionFor(reply)
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
				// ended means, we will send a streamed message
				// where we mark the message as the last one in the stream
				// and therefore, the streamed command has ended.
				delete(ctx.streamedCtx, ctx.CommandId) // TODO: do we race here somehow?
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
