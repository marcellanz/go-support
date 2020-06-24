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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const snapshotEveryDefault = 100

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

// NewServer returns an initialized Server
func NewServer() *Server {
	return &Server{
		entities: make(map[ServiceName]*Entity),
	}
}

func (s *Server) Register(e *Entity) error {
	if e.EntityFunc == nil {
		return fmt.Errorf("the entity has to define an EntityFunc but did not")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.entities[e.ServiceName]; exists {
		return fmt.Errorf("an entity with service name: %s is already registered", e.ServiceName)
	}
	e.SnapshotEvery = snapshotEveryDefault
	s.entities[e.ServiceName] = e
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
// Error handling is done so that any error returned, triggers the stream to be closed.
// If an error is a client failure, a ClientAction_Failure is sent with a command id set
// if provided by the error. If an error is a protocol failure or any other error, a
// EventSourcedStreamOut_Failure is sent. A protocol failure might provide a command id to
// be included.
func (s *Server) Handle(stream protocol.EventSourced_HandleServer) error {
	defer func() {
		if r := recover(); r != nil {
			// on panic we try to tell the proxy and panic again.
			_ = sendFailure(fmt.Errorf("Server.Handle panic-ked with: %v", r), stream)
			// there are two ways to do this
			// a) report and close the stream and let others run
			// b) report and panic and therefore crash the program
			// how can we decide that the panic keeps the user function in
			// a consistent state. this one occasion could be perfectly ok
			// to crash, but thousands of other keep running. why get them all down?
			// so then there is the presumption that a panic is truly exceptional
			// and we can't be sure about anything to be safe after one.
			// the proxy is well prepared for this, it is able to re-establish state
			// and also isolate the erroneous entity type from others.
			panic(r)
		}
	}()

	// for any error we get, we send a protocol.Failure and close the stream.
	if err := s.handle(stream); err != nil {
		if status.Code(err) == codes.Canceled {
			return err
		}
		log.Print(err)
		if sendErr := sendFailure(err, stream); sendErr != nil {
			log.Print(sendErr)
		}
		return status.Error(codes.Aborted, err.Error())
	}
	return nil
}

func (s *Server) handle(server protocol.EventSourced_HandleServer) error {
	first, err := server.Recv()
	if err == io.EOF {
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
		if runner.context.failed != nil {
			// failed means deactivated. we may never get this far.
			// context.failed should have been sent as a client reply failure
			return fmt.Errorf("failed context was not reported: %w", runner.context.failed)
		}
		if runner.context.active == false {
			return nil // TODO: what do we report here
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
			err := runner.handleCommand(m.Command)
			if err == nil {
				continue
			}
			switch t := err.(type) {
			case *protocol.ProtocolFailure:
				t.F.CommandId = m.Command.GetId()
			case *protocol.ClientFailure:
				t.F.CommandId = m.Command.GetId()
			}
			if err := sendFailure(err, runner.stream); err != nil {
				return err
			}
		case *protocol.EventSourcedStreamIn_Event:
			if err := runner.handleEvent(m.Event); err != nil {
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
		EntityId: id,
		EntityInstance: &EntityInstance{
			EventSourcedEntity: entity,
			Instance:           entity.EntityFunc(id),
		},
		EventEmitter: NewEmitter(),
		active:       true,
		// TODO: preserve context here? r.stream.Context()
		eventSequence: 0,
	}
	if snapshot := init.GetSnapshot(); snapshot != nil {
		if err := s.handleInitSnapshot(snapshot, r); err != nil {
			return err
		}
	}
	r.subscribeEvents()
	return nil
}

func (s *Server) handleInitSnapshot(snapshot *protocol.EventSourcedSnapshot, r *runner) error {
	val, err := s.unmarshalSnapshot(snapshot)
	if val == nil || err != nil {
		return &protocol.ProtocolFailure{
			Err: fmt.Errorf("handling snapshot failed with: %w", err),
		}
	}

	err = r.context.EntityInstance.EventSourcedEntity.SnapshotHandlerFunc(
		r.context.EntityInstance.Instance,
		r.context,
		val,
	)
	if err != nil {
		return &protocol.ProtocolFailure{
			Err: fmt.Errorf("handling snapshot failed with: %w", err),
		}
	}
	r.context.EntityInstance.eventSequence = snapshot.SnapshotSequence
	return nil
}

func (*Server) unmarshalSnapshot(s *protocol.EventSourcedSnapshot) (interface{}, error) {
	// see: https://developers.google.com/protocol-buffers/docs/reference/csharp/class/google/protobuf/well-known-types/any#typeurl
	typeUrl := s.Snapshot.GetTypeUrl()
	if !strings.Contains(typeUrl, "://") {
		typeUrl = "https://" + typeUrl
	}
	typeURL, err := url.Parse(typeUrl)
	if err != nil {
		return nil, err
	}
	switch typeURL.Host {
	case encoding.PrimitiveTypeURLPrefix:
		snapshot, err := encoding.UnmarshalPrimitive(s.Snapshot)
		if err != nil {
			return nil, &protocol.ProtocolFailure{
				Err: fmt.Errorf("unmarshalling snapshot failed with: %w", err),
			}
		}
		return snapshot, nil
	case encoding.ProtoAnyBase:
		msgName := strings.TrimPrefix(s.Snapshot.GetTypeUrl(), encoding.ProtoAnyBase+"/") // TODO: this might be something else than a proto message
		messageType := proto.MessageType(msgName)
		if messageType.Kind() != reflect.Ptr {
			break
		}
		if message, ok := reflect.New(messageType.Elem()).Interface().(proto.Message); ok {
			if err := proto.Unmarshal(s.Snapshot.Value, message); err != nil {
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

func (r *runner) handleEvent(event *protocol.EventSourcedEvent) error {
	if r.context == nil {
		return &protocol.ProtocolFailure{
			Err: fmt.Errorf("no entity registered with entityId: %v", r.context.EntityId),
		}
	}
	if r.context.EntityId == "" {
		return &protocol.ProtocolFailure{
			Err: fmt.Errorf("no entityId was found from a previous init message for event sequence: %v", event.Sequence),
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
func (r *runner) handleCommand(cmd *protocol.Command) error {
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
