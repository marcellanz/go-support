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

func (*runner) handleSnapshots(i *EntityInstance) (*any.Any, error) {
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

func (r *runner) handleEvents(i *EntityInstance, events ...*protocol.EventSourcedEvent) error {
	if i.EventSourcedEntity.EventFunc == nil {
		return nil
	}
	for _, event := range events {
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
		err := proto.Unmarshal(event.Payload.Value, message)
		if err != nil {
			return fmt.Errorf("%s, %w", err, encoding.ErrMarshal)
		}
		// we're ready to handle the proto message
		err = i.EventSourcedEntity.EventFunc(i.Instance, r.context, message)
		if err != nil {
			return err
		}
		if r.context.failed != nil {
			// TODO: we send a client failure
		}
	}
	return nil
}

// applyEvent applies an event to a local entity
func (r *runner) applyEvent(i *EntityInstance, event interface{}) error {
	payload, err := encoding.MarshalAny(event)
	if err != nil {
		return err
	}
	return r.handleEvents(i, &protocol.EventSourcedEvent{Payload: payload})
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
