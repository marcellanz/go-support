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

package eventsourced

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"reflect"
	"strings"
	"sync"

	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
)

const snapshotEveryDefault = 100

// marshalEventsAny receives the events emitted through the handling of a command
// and marshals them to the event serialized form.
func MarshalEventsAny(entityContext *Context) ([]*any.Any, error) {
	events := make([]*any.Any, 0)
	for _, evt := range entityContext.Events() {
		event, err := encoding.MarshalAny(evt)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	entityContext.Clear()
	return events, nil
}

// Entity captures an Entity, its ServiceName and PersistenceID.
// It is used to be registered as an event sourced entity on a CloudState instance.
type Entity struct {
	// ServiceName is the fully qualified name of the service that implements this entities interface.
	// Setting it is mandatory.
	ServiceName ServiceName
	// PersistenceID is used to namespace events in the journal, useful for
	// when you share the same database between multiple entities. It defaults to
	// the simple name for the entity type.
	// It’s good practice to select one explicitly, this means your database
	// isn’t depend on type names in your code.
	// Setting it is mandatory.
	PersistenceID string
	// The snapshotEvery parameter controls how often snapshots are taken,
	// so that the entity doesn't need to be recovered from the whole journal
	// each time it’s loaded. If left unset, it defaults to 100.
	// Setting it to a negative number will result in snapshots never being taken.
	SnapshotEvery int64

	// EntityFactory is a factory method which generates a new Entity.
	EntityFunc func(id EntityId) interface{}

	CommandFunc         func(entity interface{}, ctx *Context, name string, msg proto.Message) (reply proto.Message, err error)
	SnapshotHandlerFunc func(entity interface{}, ctx *Context, snapshot interface{}) error
	SnapshotFunc        func(entity interface{}) (snapshot interface{}, err error)
	EventFunc           func(entity interface{}, ctx *Context, event interface{}) error
}

// Server is the implementation of the Server server API for EventSourced service.
type Server struct {
	// mu protects the map below.
	mu sync.RWMutex
	// entities are indexed by their service name.
	entities map[ServiceName]*Entity
}

// newEventSourcedServer returns an initialized Server
func NewServer() *Server {
	return &Server{
		entities: make(map[ServiceName]*Entity),
	}
}

func (s *Server) Register(ese *Entity) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.entities[ese.ServiceName]; exists {
		return fmt.Errorf("an entity with service name: %s is already registered", ese.ServiceName)
	}
	ese.SnapshotEvery = snapshotEveryDefault
	s.entities[ese.ServiceName] = ese
	return nil
}

// Handle handles the stream. One stream will be established per active entity.
// Once established, the first message sent will be Init, which contains the entity ID, and,
// if the entity has previously persisted a snapshot, it will contain that snapshot. It will
// then send zero to many event messages, one for each event previously persisted. The entity
// is expected to apply these to its state in a deterministic fashion. Once all the events
// are sent, one to many commands are sent, with new commands being sent as new requests for
// the entity come in. The entity is expected to reply to each command with exactly one reply
// message. The entity should reply in order, and any events that the entity requests to be
// persisted the entity should handle itself, applying them to its own state, as if they had
// arrived as events when the event stream was being replayed on load.
//
// Error handling is done, so that any error returned triggers the stream to be closed.
// If an error is a client failure, a ClientAction_Failure is sent with a command id set
// if provided by the error. If an error is a protocol failure or any other error, a
// EventSourcedStreamOut_Failure is sent. A protocol failure might provide a command id to
// be included.
func (s *Server) Handle(stream protocol.EventSourced_HandleServer) (handleErr error) {
	defer func() {
		if r := recover(); r != nil {
			// tell the proxy, and panic
			handleErr = sendFailure(fmt.Errorf("Server.Handle panic-ked with: %v", r), stream)
			panic(r)
		}
	}()
	if err := s.handle(stream); err != nil {
		log.Print(err)
		return sendFailure(err, stream)
	}
	return nil
}

func (s *Server) handle(server protocol.EventSourced_HandleServer) error {
	first, err := server.Recv()
	if err == io.EOF { // the stream has ended
		return nil
	}
	if err != nil {
		return err
	}
	runner := &runner{stream: server}
	switch m := first.GetMessage().(type) {
	case *protocol.EventSourcedStreamIn_Init:
		if err := s.handleInit(m.Init, runner); err != nil {
			return err
		}
	default:
		return fmt.Errorf("a message was received without having a EventSourcedInit message handled before: %v", first.GetMessage())
	}

	for {
		select {
		case <-runner.stream.Context().Done():
			return runner.stream.Context().Err()
		default:
		}
		if runner.context.failed != nil {
			// failed means deactivated. we may never get this far.
			return nil
		}
		msg, err := server.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch m := msg.GetMessage().(type) {
		case *protocol.EventSourcedStreamIn_Command:
			err := s.handleCommand(m.Command, runner)
			if err == nil {
				continue
			}
			switch t := err.(type) {
			case *protocol.ProtocolFailure:
				t.F.CommandId = m.Command.Id
			case *protocol.ClientFailure:
				t.F.CommandId = m.Command.Id
			}
			if err := sendFailure(err, runner.stream); err != nil {
				return err
			}
		case *protocol.EventSourcedStreamIn_Event:
			// TODO spec: Why does command carry the entityId and an event not?
			if err := s.handleEvent(m.Event, runner); err != nil {
				return err
			}
		case *protocol.EventSourcedStreamIn_Init:
			return errors.New("duplicate init message for the same entity")
		case nil:
			return errors.New("empty message received")
		default:
			return fmt.Errorf("unknown message received: %v", msg.GetMessage())
		}
	}
}

func (s *Server) handleInit(init *protocol.EventSourcedInit, r *runner) error {
	id := EntityId(init.GetEntityId())
	s.mu.RLock()
	entity := s.entities[ServiceName(init.GetServiceName())]
	s.mu.RUnlock()

	r.context = &Context{
		EntityId: id,
		EntityInstance: &EntityInstance{
			EventSourcedEntity: entity,
			Instance:           entity.EntityFunc(id),
		},
		EventEmitter:  NewEmitter(),
		eventSequence: 0,
		active:        true,
	}

	if err := s.handleInitSnapshot(init, r); err != nil {
		return fmt.Errorf("unable to server.Send: %w", err)
	}
	r.subscribeEvents()
	return nil
}

func (s *Server) handleInitSnapshot(init *protocol.EventSourcedInit, r *runner) error {
	if init.Snapshot == nil {
		return nil
	}
	snapshot, err := s.unmarshalSnapshot(init)
	if snapshot == nil || err != nil {
		return &protocol.ProtocolFailure{
			Err: fmt.Errorf("handling snapshot failed with: %w", err),
		}
	}

	err = r.context.EntityInstance.EventSourcedEntity.SnapshotHandlerFunc(
		r.context.EntityInstance.Instance,
		r.context,
		snapshot,
	)
	if err != nil {
		return &protocol.ProtocolFailure{
			Err: fmt.Errorf("handling snapshot failed with: %w", err),
		}
	}
	r.context.EntityInstance.eventSequence = init.GetSnapshot().SnapshotSequence
	return nil
}

func (Server) unmarshalSnapshot(init *protocol.EventSourcedInit) (interface{}, error) {
	// see: https://developers.google.com/protocol-buffers/docs/reference/csharp/class/google/protobuf/well-known-types/any#typeurl
	typeUrl := init.Snapshot.Snapshot.GetTypeUrl()
	if !strings.Contains(typeUrl, "://") {
		typeUrl = "https://" + typeUrl
	}
	typeURL, err := url.Parse(typeUrl)
	if err != nil {
		return nil, err
	}
	switch typeURL.Host {
	case encoding.PrimitiveTypeURLPrefix:
		snapshot, err := encoding.UnmarshalPrimitive(init.Snapshot.Snapshot)
		if err != nil {
			return nil, &protocol.ProtocolFailure{
				Err: fmt.Errorf("unmarshalling snapshot failed with: %w", err),
			}
		}
		return snapshot, nil
	case encoding.ProtoAnyBase:
		msgName := strings.TrimPrefix(init.Snapshot.Snapshot.GetTypeUrl(), encoding.ProtoAnyBase+"/") // TODO: this might be something else than a proto message
		messageType := proto.MessageType(msgName)
		if messageType.Kind() != reflect.Ptr {
			break
		}
		if message, ok := reflect.New(messageType.Elem()).Interface().(proto.Message); ok {
			if err := proto.Unmarshal(init.Snapshot.Snapshot.Value, message); err != nil {
				return nil, &protocol.ProtocolFailure{
					Err: fmt.Errorf("unmarshalling snapshot failed with: %w", err),
				}
			}
			return message, nil
		}
	}
	return nil, &protocol.ProtocolFailure{
		Err: fmt.Errorf("unmarshalling snapshot failed with: no snapshot unmarshaller found for: %v", typeURL.String()),
	}
}

func (s *Server) handleEvent(event *protocol.EventSourcedEvent, r *runner) error {
	if r.context.EntityId == "" {
		return &protocol.ProtocolFailure{
			Err: fmt.Errorf("no entityId was found from a previous init message for event sequence: %v", event.Sequence),
		}
	}
	if r.context == nil {
		return &protocol.ProtocolFailure{
			Err: fmt.Errorf("no entity with entityId registered: %v", r.context.EntityId),
		}
	}
	if err := r.handleEvents(r.context.EntityInstance, event); err != nil {
		return &protocol.ProtocolFailure{
			Err: fmt.Errorf("handle event failed: %v", err),
		}
	}
	return nil
}

// handleCommand handles a command received from the Cloudstate proxy.
//
// TODO: remove these following lines of comment
// "Unary RPCs where the client sends a single request to the server and
// gets a single response back, just like a normal function call." are supported right now.
//
// to handle a command we need
// - the entity id, which identifies the entity (its instance) uniquely(?) for this user function instance
// - the service name, like "com.example.shoppingcart.ShoppingCart"
// - a command id
// - a command name, which is one of the gRPC service rpcs defined by this entities service
// - the command payload, which is the message sent for the command as a protobuf.Any blob
// - a streamed flag, (TODO: for what?)
//
// together, these properties allow to call a method of the entities registered service and
// return its response as a reply to the Cloudstate proxy.
//
// Events:
// Beside calling the service method, we have to collect "events" the service might emit.
// These events afterwards have to be handled by a EventHandler to update the state of the
// entity. The Cloudstate proxy can re-play these events at any time
func (s *Server) handleCommand(cmd *protocol.Command, r *runner) error {
	msgName := strings.TrimPrefix(cmd.Payload.GetTypeUrl(), encoding.ProtoAnyBase+"/")
	messageType := proto.MessageType(msgName)
	if messageType.Kind() != reflect.Ptr {
		return fmt.Errorf("messageType: %s is of non Ptr kind", messageType)
	}
	// get a zero-ed message of this type
	message, ok := reflect.New(messageType.Elem()).Interface().(proto.Message)
	if !ok {
		return fmt.Errorf("messageType is no proto.Message: %v", messageType)
	}
	// and marshal onto it what we got as an any.Any onto it
	err := proto.Unmarshal(cmd.Payload.Value, message)
	if err != nil {
		return fmt.Errorf("%s, %w", err, encoding.ErrMarshal)
	}

	// The gRPC implementation returns the rpc return method
	// and an error as a second return value.
	reply, errReturned := r.context.EntityInstance.EventSourcedEntity.CommandFunc(
		r.context.EntityInstance.Instance, r.context, cmd.Name, message,
	)
	// the error
	if errReturned != nil {
		// TCK says: TODO Expects entity.Failure, but gets clientAction.Action.Failure(Failure(commandId, msg)))
		return &protocol.ProtocolFailure{
			F:   &protocol.Failure{CommandId: cmd.GetId()},
			Err: errReturned,
		}
	}
	// context failed
	if r.context.failed != nil {
		cf := &protocol.ClientFailure{
			F:   &protocol.Failure{CommandId: cmd.GetId()},
			Err: r.context.failed,
		}
		r.context.failed = nil
		return cf
	}
	// the reply
	callReply, err := encoding.MarshalAny(reply)
	if err != nil { // this should never happen
		return &protocol.ProtocolFailure{
			F:   &protocol.Failure{CommandId: cmd.GetId()},
			Err: fmt.Errorf("called return value at index 0 is no proto.Message. %w", err),
		}
	}
	// emitted events
	events, err := MarshalEventsAny(r.context)
	if err != nil {
		return &protocol.ProtocolFailure{F: &protocol.Failure{CommandId: cmd.GetId()}, Err: err}
	}
	// snapshot
	snapshot, err := r.handleSnapshots(r.context.EntityInstance)
	if err != nil {
		return &protocol.ProtocolFailure{F: &protocol.Failure{CommandId: cmd.GetId()}, Err: err}
	}
	sourcedReply := &protocol.EventSourcedReply{
		CommandId: cmd.GetId(),
		ClientAction: &protocol.ClientAction{
			Action: &protocol.ClientAction_Reply{
				Reply: &protocol.Reply{
					Payload: callReply,
				},
			},
		},
		Events:   events,
		Snapshot: snapshot,
	}
	return sendEventSourcedReply(sourcedReply, r.stream)
}

type runner struct {
	stream        protocol.EventSourced_HandleServer
	context       *Context
	stateReceived bool
}

func (r *runner) subscribeEvents() {
	r.context.Subscribe(&Subscription{
		OnNext: func(event interface{}) error {
			if err := r.applyEvent(r.context.EntityInstance, event); err != nil {
				return err
			}
			r.context.eventSequence++
			return nil
		},
		OnErr: func(err error) {
			r.context.Failed(err)
		},
	})
}

func (r *runner) handleSnapshots(i *EntityInstance) (*any.Any, error) {
	if !i.shouldSnapshot() {
		return nil, nil
	}
	snap, err := i.EventSourcedEntity.SnapshotFunc(i.Instance)
	if err != nil {
		return nil, fmt.Errorf("getting a snapshot has failed: %v. %w", err, protocol.ErrProtocolFailure)
	}
	// TODO: we expect a proto.Message but should support other formats
	snapshot, err := encoding.MarshalAny(snap)
	if err != nil {
		return nil, err
	}
	i.resetSnapshotEvery()
	return snapshot, nil
}

// applyEvent applies an event to a local entity
func (r *runner) applyEvent(entityInstance *EntityInstance, event interface{}) error {
	payload, err := encoding.MarshalAny(event)
	if err != nil {
		return err
	}
	return r.handleEvents(entityInstance, &protocol.EventSourcedEvent{Payload: payload})
}

func (r *runner) handleEvents(entityInstance *EntityInstance, events ...*protocol.EventSourcedEvent) error {
	if entityInstance.EventSourcedEntity.EventFunc == nil {
		return nil
	}
	for _, event := range events {
		// TODO: here's the point where events can be protobufs, serialized as json or other formats
		msgName := strings.TrimPrefix(event.Payload.GetTypeUrl(), encoding.ProtoAnyBase+"/")
		messageType := proto.MessageType(msgName)
		if messageType.Kind() != reflect.Ptr {
			continue
		}
		// get a zero-ed message of this type
		message, ok := reflect.New(messageType.Elem()).Interface().(proto.Message)
		if !ok {
			continue
		}
		// and marshal what we got as an any.Any onto it
		err := proto.Unmarshal(event.Payload.Value, message)
		if err != nil {
			return fmt.Errorf("%s, %w", err, encoding.ErrMarshal)
		}
		// we're ready to handle the proto message
		err = entityInstance.EventSourcedEntity.EventFunc(entityInstance.Instance, r.context, message)
		if err != nil {
			return err // FIXME/TODO: is this correct? if we fail here, nothing is safe afterwards.
		}
	} // TODO: what do we do if we haven't handled the events?
	return nil
}
