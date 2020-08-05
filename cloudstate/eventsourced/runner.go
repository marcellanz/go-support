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
	"net/url"
	"reflect"
	"strings"

	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
)

type runner struct {
	stream        protocol.EventSourced_HandleServer
	context       *Context
	stateReceived bool
}

func (r *runner) handleSnapshot() (*any.Any, error) {
	if !r.context.shouldSnapshot() {
		return nil, nil
	}
	s, ok := r.context.Instance.(Snapshooter)
	if !ok {
		return nil, nil
	}
	snap, err := s.Snapshot(r.context)
	if err != nil {
		return nil, fmt.Errorf("getting a snapshot has failed: %w", err)
	}
	// TODO: we expect a proto.Message but should support other formats
	snapshot, err := encoding.MarshalAny(snap)
	if err != nil {
		return nil, err
	}
	r.context.resetSnapshotEvery()
	return snapshot, nil
}

func (r *runner) handleEvent(event *protocol.EventSourcedEvent) error {
	// TODO: here's the point where events can be protobufs, serialized as json or other formats
	msgName := strings.TrimPrefix(event.Payload.GetTypeUrl(), encoding.ProtoAnyBase+"/")
	messageType := proto.MessageType(msgName)
	if messageType.Kind() != reflect.Ptr {
		return fmt.Errorf("messageType.Kind() is not a pointer type: %v", messageType)
	}
	// get a zero-ed message of this type
	message, ok := reflect.New(messageType.Elem()).Interface().(proto.Message)
	if !ok {
		return fmt.Errorf("unable to create a new zero-ed message of type: %v", messageType)
	}
	// and marshal what we got as an any.Any onto it
	if err := proto.Unmarshal(event.Payload.Value, message); err != nil {
		return fmt.Errorf("%s, %w", err, encoding.ErrMarshal)
	}
	// we're ready to handle the proto message
	if err := r.context.Instance.HandleEvent(r.context, message); err != nil {
		return err
	}
	return r.context.failed
}

// applyEvent applies an event to a local entity.
func (r *runner) applyEvent(event interface{}) error {
	payload, err := encoding.MarshalAny(event)
	if err != nil {
		return err
	}
	return r.handleEvent(&protocol.EventSourcedEvent{Payload: payload})
}

func (r *runner) handleInitSnapshot(snapshot *protocol.EventSourcedSnapshot) error {
	val, err := r.unmarshalSnapshot(snapshot)
	if val == nil || err != nil {
		return fmt.Errorf("handling snapshot failed with: %w", err)
	}
	s, ok := r.context.Instance.(Snapshooter)
	if !ok {
		return fmt.Errorf("entity instance does not implement eventsourced.Snapshooter")
	}
	err = s.HandleSnapshot(r.context, val)
	if err != nil {
		return fmt.Errorf("handling snapshot failed with: %w", err)
	}
	r.context.eventSequence = snapshot.SnapshotSequence
	return nil
}

func (*runner) unmarshalSnapshot(s *protocol.EventSourcedSnapshot) (interface{}, error) {
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
			return nil, err
		}
		return snapshot, nil
	case encoding.ProtoAnyBase:
		msgName := strings.TrimPrefix(s.Snapshot.GetTypeUrl(), encoding.ProtoAnyBase+"/") // TODO: this might be something else than a proto message
		messageType := proto.MessageType(msgName)
		if messageType.Kind() != reflect.Ptr {
			return nil, err
		}
		message, ok := reflect.New(messageType.Elem()).Interface().(proto.Message)
		if !ok {
			return nil, err
		}
		if err := proto.Unmarshal(s.Snapshot.Value, message); err != nil {
			return nil, err
		}
		return message, nil
	}
	return nil, fmt.Errorf("no snapshot unmarshaller found for: %v", typeURL.String())
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
// entity. The Cloudstate proxy can re-play these events at any time.
func (r *runner) handleCommand(cmd *protocol.Command) error {
	msgName := strings.TrimPrefix(cmd.Payload.GetTypeUrl(), encoding.ProtoAnyBase+"/")
	messageType := proto.MessageType(msgName)
	if messageType.Kind() != reflect.Ptr {
		return fmt.Errorf("messageType: %s is of non Ptr kind", messageType)
	}
	// get a zero-ed message of this type.
	message, ok := reflect.New(messageType.Elem()).Interface().(proto.Message)
	if !ok {
		return fmt.Errorf("messageType is no proto.Message: %v", messageType)
	}
	// and unmarshal it.
	err := proto.Unmarshal(cmd.Payload.Value, message)
	if err != nil {
		return fmt.Errorf("%s, %w", err, encoding.ErrMarshal)
	}
	// The gRPC implementation returns the rpc return method and an error as a second return value.
	reply, errReturned := r.context.Instance.HandleCommand(r.context, cmd.Name, message)
	if errReturned != nil {
		return r.sendClientActionFailure(&protocol.Failure{
			CommandId:   cmd.Id,
			Description: errReturned.Error(),
		})
	}
	// the context may have failed. As it is not defined what state
	// a user function would have after a client failure, we get safe and
	// let the stream fail.
	if r.context.failed != nil {
		return r.context.failed
	}
	// the reply
	callReply, err := encoding.MarshalAny(reply)
	if err != nil { // this should never happen
		return protocol.ServerError{
			Failure: &protocol.Failure{CommandId: cmd.GetId()},
			Err:     fmt.Errorf("marshalling of reply failed. %w", err),
		}
	}
	// emitted events
	events, err := r.context.marshalEventsAny()
	if err != nil {
		return protocol.ServerError{
			Failure: &protocol.Failure{CommandId: cmd.GetId()},
			Err:     err,
		}
	}
	// snapshot
	snapshot, err := r.handleSnapshot()
	if err != nil {
		return protocol.ServerError{
			Failure: &protocol.Failure{CommandId: cmd.GetId()},
			Err:     err,
		}
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
	return r.sendEventSourcedReply(sourcedReply)
}

func (r *runner) sendEventSourcedReply(rep *protocol.EventSourcedReply) error {
	return r.stream.Send(&protocol.EventSourcedStreamOut{
		Message: &protocol.EventSourcedStreamOut_Reply{
			Reply: rep,
		},
	})
}

func (r *runner) sendClientActionFailure(f *protocol.Failure) error {
	return r.sendEventSourcedReply(&protocol.EventSourcedReply{
		CommandId: f.CommandId,
		ClientAction: &protocol.ClientAction{
			Action: &protocol.ClientAction_Failure{
				Failure: f,
			},
		},
	})
}
