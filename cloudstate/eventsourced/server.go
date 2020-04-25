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
	"fmt"
	"io"
	"net/url"
	"reflect"
	"strings"

	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
)

// marshalEventsAny receives the events emitted through the handling of a command
// and marshals them to the event serialized form.
func MarshalEventsAny(entityContext *EntityInstanceContext) ([]*any.Any, error) {
	events := make([]*any.Any, 0)
	for _, evt := range entityContext.context.Events() {
		event, err := encoding.MarshalAny(evt)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	entityContext.context.Clear()
	return events, nil
}

const snapshotEveryDefault = 100

// EventSourcedEntity captures an Entity, its ServiceName and PersistenceID.
// It is used to be registered as an event sourced entity on a CloudState instance.
type EventSourcedEntity struct {
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

// EventSourcedServer is the implementation of the EventSourcedServer server API for EventSourced service.
type EventSourcedServer struct {
	// entities are indexed by their service name
	entities map[ServiceName]*EventSourcedEntity
	// contexts are entity instance contexts indexed by their entity ids
	contexts map[string]*EntityInstanceContext
}

// newEventSourcedServer returns an initialized EventSourcedServer
func NewServer() *EventSourcedServer {
	return &EventSourcedServer{
		entities: make(map[ServiceName]*EventSourcedEntity),
		contexts: make(map[string]*EntityInstanceContext),
	}
}

func (esh *EventSourcedServer) Register(ese *EventSourcedEntity) error {
	if _, exists := esh.entities[ese.ServiceName]; exists {
		return fmt.Errorf("EventSourcedEntity with service name: %s is already registered", ese.ServiceName)
	}
	esh.entities[ese.ServiceName] = ese
	ese.SnapshotEvery = snapshotEveryDefault
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
func (esh *EventSourcedServer) Handle(stream protocol.EventSourced_HandleServer) error {
	var entityId string
	var failed error
	for {
		if failed != nil {
			return failed
		}
		msg, recvErr := stream.Recv()
		if recvErr == io.EOF {
			return nil
		}
		if recvErr != nil {
			return recvErr
		}
		if cmd := msg.GetCommand(); cmd != nil {
			if err := esh.handleCommand(cmd, stream); err != nil {
				// TODO: in general, what happens with the stream here if an error happens?
				failed = handleFailure(err, stream, cmd.GetId())
			}
			continue
		}
		if event := msg.GetEvent(); event != nil {
			// TODO spec: Why does command carry the entityId and an event not?
			if err := esh.handleEvent(entityId, event); err != nil {
				failed = handleFailure(err, stream, 0)
			}
			continue
		}
		if init := msg.GetInit(); init != nil {
			if err := esh.handleInit(init); err != nil {
				failed = handleFailure(err, stream, 0)
			}
			entityId = init.GetEntityId()
			continue
		}
	}
}

func (esh *EventSourcedServer) handleInit(init *protocol.EventSourcedInit) error {
	id := init.GetEntityId()
	if _, present := esh.contexts[id]; present {
		return fmt.Errorf("unable to server.Send")
	}
	entity := esh.entities[ServiceName(init.GetServiceName())]
	esh.contexts[id] = &EntityInstanceContext{
		EntityInstance: &EntityInstance{
			Instance:           entity.EntityFunc(EntityId(id)),
			EventSourcedEntity: entity,
		},
		active: true,
	}
	// TODO: move up
	ctx := &Context{
		EntityId:     EntityId(id),
		Entity:       entity,
		Instance:     esh.contexts[id].EntityInstance.Instance,
		EventEmitter: NewEmitter(),
	}
	esh.contexts[id].context = ctx

	if err := esh.handleInitSnapshot(ctx, init); err != nil {
		return fmt.Errorf("unable to server.Send: %w", err)
	}
	esh.subscribeEvents(ctx, esh.contexts[id].EntityInstance)
	return nil
}

func (esh *EventSourcedServer) handleInitSnapshot(ctx *Context, init *protocol.EventSourcedInit) error {
	if init.Snapshot == nil {
		return nil
	}
	entityId := init.GetEntityId()

	snapshot, err := esh.unmarshalSnapshot(init)
	if snapshot == nil || err != nil {
		return NewFailureErrorf("handling snapshot failed with: %v", err)
	}
	err = esh.contexts[entityId].EntityInstance.EventSourcedEntity.SnapshotHandlerFunc(
		esh.contexts[entityId].EntityInstance.Instance,
		ctx,
		snapshot,
	)
	if err != nil {
		return NewFailureErrorf("handling snapshot failed with: %v", err)
	}
	esh.contexts[entityId].EntityInstance.eventSequence = init.GetSnapshot().SnapshotSequence
	return nil
}

func (EventSourcedServer) unmarshalSnapshot(init *protocol.EventSourcedInit) (interface{}, error) {
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
			return nil, fmt.Errorf("unmarshalling snapshot failed with: %v", err)
		}
		return snapshot, nil
	case encoding.ProtoAnyBase:
		msgName := strings.TrimPrefix(init.Snapshot.Snapshot.GetTypeUrl(), encoding.ProtoAnyBase+"/") // TODO: this might be something else than a proto message
		messageType := proto.MessageType(msgName)
		if messageType.Kind() == reflect.Ptr {
			if message, ok := reflect.New(messageType.Elem()).Interface().(proto.Message); ok {
				err := proto.Unmarshal(init.Snapshot.Snapshot.Value, message)
				if err != nil {
					return nil, fmt.Errorf("unmarshalling snapshot failed with: %v", err)
				}
				return message, nil
			}
		}
	}
	return nil, fmt.Errorf("unmarshalling snapshot failed with: no snapshot unmarshaller found for: %v", typeURL.String())
}

func (esh *EventSourcedServer) subscribeEvents(ctx *Context, instance *EntityInstance) {
	ctx.Subscribe(&Subscription{
		OnNext: func(event interface{}) error {
			err := esh.applyEvent(instance, event)
			if err == nil {
				instance.eventSequence++
			}
			return err
		},
		OnErr: func(err error) {
			ctx.Failed(err)
		},
	})
}

func (esh *EventSourcedServer) handleEvent(entityId string, event *protocol.EventSourcedEvent) error {
	if entityId == "" {
		return NewFailureErrorf("no entityId was found from a previous init message for event sequence: %v", event.Sequence)
	}
	entityContext := esh.contexts[entityId]
	if entityContext == nil {
		return NewFailureErrorf("no entity with entityId registered: %v", entityId)
	}
	err := esh.handleEvents(entityContext.context, entityContext.EntityInstance, event)
	if err != nil {
		return NewFailureErrorf("handle event failed: %v", err)
	}
	return err
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
func (esh *EventSourcedServer) handleCommand(cmd *protocol.Command, server protocol.EventSourced_HandleServer) error {
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

	// we're ready to handle the proto message
	entityContext := esh.contexts[cmd.GetEntityId()]

	// The gRPC implementation returns the rpc return method
	// and an error as a second return value.
	reply, errReturned := entityContext.EntityInstance.EventSourcedEntity.CommandFunc(
		entityContext.EntityInstance.Instance,
		entityContext.context,
		cmd.Name,
		message,
	)
	// the error
	if errReturned != nil {
		// TCK says: TODO Expects entity.Failure, but gets clientAction.Action.Failure(Failure(commandId, msg)))
		return NewProtocolFailure(protocol.Failure{
			CommandId:   cmd.GetId(),
			Description: errReturned.Error(),
		})
	}
	// the reply
	callReply, err := encoding.MarshalAny(reply)
	if err != nil { // this should never happen
		return NewProtocolFailure(protocol.Failure{
			CommandId:   cmd.GetId(),
			Description: fmt.Errorf("called return value at index 0 is no proto.Message. %w", err).Error(),
		})
	}
	// emitted events
	events, err := MarshalEventsAny(entityContext)
	if err != nil {
		return NewProtocolFailure(protocol.Failure{
			CommandId:   cmd.GetId(),
			Description: err.Error(),
		})
	}
	// snapshot
	snapshot, err := esh.handleSnapshots(entityContext)
	if err != nil {
		return NewProtocolFailure(protocol.Failure{
			CommandId:   cmd.GetId(),
			Description: err.Error(),
		})
	}
	return sendEventSourcedReply(&protocol.EventSourcedReply{
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
	}, server)
}

func (*EventSourcedServer) handleSnapshots(entityContext *EntityInstanceContext) (*any.Any, error) {
	if !entityContext.EntityInstance.shouldSnapshot() {
		return nil, nil
	}
	snap, err := entityContext.EntityInstance.EventSourcedEntity.SnapshotFunc(entityContext.EntityInstance.Instance)
	if err != nil {
		return nil, fmt.Errorf("getting a snapshot has failed: %v. %w", err, ErrFailure)
	}
	// TODO: we expect a proto.Message but should support other formats
	snapshot, err := encoding.MarshalAny(snap)
	if err != nil {
		return nil, err
	}
	entityContext.EntityInstance.resetSnapshotEvery()
	return snapshot, nil
}

// applyEvent applies an event to a local entity
func (esh EventSourcedServer) applyEvent(entityInstance *EntityInstance, event interface{}) error {
	payload, err := encoding.MarshalAny(event)
	if err != nil {
		return err
	}
	return esh.handleEvents(nil, entityInstance, &protocol.EventSourcedEvent{Payload: payload})
}

func (EventSourcedServer) handleEvents(ctx *Context, entityInstance *EntityInstance, events ...*protocol.EventSourcedEvent) error {
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
		// and marshal onto it what we got as an any.Any onto it
		err := proto.Unmarshal(event.Payload.Value, message)
		if err != nil {
			return fmt.Errorf("%s, %w", err, encoding.ErrMarshal)
		}
		// we're ready to handle the proto message
		err = entityInstance.EventSourcedEntity.EventFunc(entityInstance.Instance, ctx, message)
		if err != nil {
			return err // FIXME/TODO: is this correct? if we fail here, nothing is safe afterwards.
		}
	} // TODO: what do we do if we haven't handled the events?
	return nil
}
