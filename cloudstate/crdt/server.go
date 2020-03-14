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

type (
	ServiceName string
	EntityId    string
)

func (sn ServiceName) String() string {
	return string(sn)
}

type Server struct {
	// entities has descriptions of entities registered by service names
	entities map[ServiceName]*Entity
	sync.RWMutex
}

// CrdtEntities can be registered to a server that handles CRDT entities by a ServiceName.
// Whenever a crdt.Server receives an CrdInit for an instance of a CRDT entity identified by its
// EntityId and a ServiceName, the crdt.Server handles such entities through their lifecycle.
// The handled entities value are captured by a context that is held fo each of them.
func (s *Server) RegisterEntity(e *Entity, serviceName ServiceName) error {
	e.registerOnce.Do(func() {
		s.Lock()
		s.entities[serviceName] = e
		s.Unlock()
	})
	return nil
}

func NewServer() *Server {
	return &Server{
		entities: make(map[ServiceName]*Entity),
	}
}

type CrdtContext struct {
}

// Entity captures an Entity with its ServiceName.
// It is used to be registered as an CRDT entity on a Cloudstate instance.
type Entity struct {
	// ServiceName is the fully qualified name of the service that implements this entities interface.
	// Setting it is mandatory.
	ServiceName ServiceName

	// DefaultFunc is a factory method which generates a new Entity.
	DefaultFunc func(cc *CrdtContext) error

	// internal
	registerOnce sync.Once
}

type runner struct {
	id            EntityId
	streamed      bool
	context       *EntityInstance
	stream        protocol.Crdt_HandleServer
	stateReceived bool
	deleted       bool
}

// After invoking handle, the first message sent will always be a CrdtInit message, containing the entity ID, and,
// if it exists or is available, the current value of the entity. After that, one or more commands may be sent,
// as well as deltas as they arrive, and the entire value if either the entity is created, or the proxy wishes the
// user function to replace its entire value.
// The user function must respond with one reply per command in. They do not necessarily have to be sent in the same
// order that the commands were sent, the command ID is used to correlate commands to replies.
func (s *Server) Handle(stream protocol.Crdt_HandleServer) error {
	first, err := stream.Recv()
	if err == io.EOF { // stream has ended
		return nil
	}
	if err != nil {
		return err
	}

	// first always a CrdtInit message must be received
	runner := &runner{
		stream: stream,
	}
	switch m := first.GetMessage().(type) {
	case *protocol.CrdtStreamIn_Init:
		if err = s.handleInit(m.Init, runner); err != nil {
			return sendFailureAndReturnWith(fmt.Errorf("handling of CrdtInit failed with: %w", err), runner.stream)
		}
		if state := m.Init.State.GetState(); state != nil {
			if err := s.handleState(m.Init.State, runner); err != nil {
				return sendFailureAndReturnWith(err, runner.stream)
			}
		}
	default:
		return sendFailureAndReturnWith(
			errors.New("a message was received without having an CrdtInit message first"), runner.stream,
		)
	}
	if runner.id == "" {
		return sendFailureAndReturnWith(
			errors.New("a message was received without having a CrdtInit message before"), runner.stream,
		)
	}

	// handle all other messages after a CrdtInit message was received
	for {
		ctx := runner.stream.Context()
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		msg, err := runner.stream.Recv()
		if err == io.EOF { // stream has ended
			return nil
		}
		if err != nil {
			return err
		}
		switch m := msg.GetMessage().(type) {
		case *protocol.CrdtStreamIn_State:
			if err := s.handleState(m.State, runner); err != nil {
				return sendFailureAndReturnWith(err, runner.stream)
			}
			runner.stateReceived = true
		case *protocol.CrdtStreamIn_Changed:
			if !runner.stateReceived {
				return sendFailureAndReturnWith(
					errors.New("received a CrdtDelta message without having a CrdtState ever received"), runner.stream,
				)
			}
			if err := s.handleDelta(m.Changed, runner); err != nil {
				return sendFailureAndReturnWith(err, runner.stream)
			}
		case *protocol.CrdtStreamIn_Deleted:
			// Delete the entity. May be sent at any time. The user function should clear its value when it receives this.
			// A proxy may decide to terminate the stream after sending this.
			runner.deleted, runner.context = true, nil
		case *protocol.CrdtStreamIn_Command:
			// A command, may be sent at any time.
			runner.streamed = m.Command.Streamed
		case *protocol.CrdtStreamIn_StreamCancelled:
			if err := handleCancellation(m.StreamCancelled, runner); err != nil {
				return sendFailureAndReturnWith(err, runner.stream)
			}
		case *protocol.CrdtStreamIn_Init:
			return sendFailureAndReturnWith(errors.New("duplicate init event for the same entity"), runner.stream)
		case nil:
			return sendFailureAndReturnWith(errors.New("empty message received"), runner.stream)
		default:
			return sendFailureAndReturnWith(errors.New("unknown message received"), runner.stream)
		}
	}
}

// A stream has been cancelled.
// A streamed command handler may also register an onCancel callback to be notified when the stream
// is cancelled. The cancellation callback handler may update the CRDT. This is useful if the CRDT
// is being used to track connections, for example, when using Vote CRDTs to track a users online status.
func handleCancellation(cancelled *protocol.StreamCancelled, r *runner) error {
	return nil
}

func (s *Server) handleInit(init *protocol.CrdtInit, r *runner) error {
	serviceName := ServiceName(init.GetServiceName())
	s.RLock()
	entity, exists := s.entities[serviceName]
	s.RUnlock()
	if !exists {
		return errors.New("received a command for an unknown CRDT service")
	}
	r.id = EntityId(init.GetEntityId())
	r.context = &EntityInstance{
		EntityId: r.id,
		Entity:   entity,
		active:   true,
	}
	return nil
}

// A CrdtState message is sent to indicate the user function should replace its current value with this value. If the user function
// does not have a current value, either because the init function didn'm send one and the user function hasn'm
// updated the value itself in response to a command, or because the value was deleted, this must be sent before
// any deltas.
func (s *Server) handleState(state *protocol.CrdtState, r *runner) error {
	switch s := state.State.(type) {
	case *protocol.CrdtState_Gcounter:
		c := &GCounter{}
		c.applyState(state)
		r.context.Instance = c
	case *protocol.CrdtState_Pncounter:
		c := &PNCounter{}
		c.applyState(s.Pncounter)
		r.context.Instance = c
	default:
		return errors.New(fmt.Sprintf("unhandled value type received: %value", state))
	}
	return nil
}

// A lwwRegisterDelta to be applied to the current value. It may be sent at any time as long as the user function already has
// value.
func (s *Server) handleDelta(delta *protocol.CrdtDelta, r *runner) error {
	switch d := delta.Delta.(type) {
	case *protocol.CrdtDelta_Gcounter:
		if c, ok := r.context.Instance.(*GCounter); ok {
			c.applyDelta(delta)
		} else {
			return errors.New(fmt.Sprintf(
				"received GcounterDelta %+value, but it doesn't match the CRDT that this entity has: %+value", d, c,
			))
		}
		//TODO notifySubscribers
	case *protocol.CrdtDelta_Pncounter:
		if c, ok := r.context.Instance.(*PNCounter); ok {
			c.applyDelta(d.Pncounter)
		} else {
			return errors.New(fmt.Sprintf(
				"received PNCounterDelta %+value, but it doesn't match the CRDT that this entity has: %+value", d, c,
			))
		}
		//TODO notifySubscribers
	default:
		return errors.New(fmt.Sprintf("unhandled value type received: %value", d))
	}
	return nil
}
