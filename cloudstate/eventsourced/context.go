//
// Copyright 2019 Lightbend Inc.
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
	"context"

	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/golang/protobuf/ptypes/any"
)

type Context struct {
	EntityId EntityId
	// EventSourcedEntity describes the instance hold by the EntityInstance.
	EventSourcedEntity *Entity
	// Instance is an instance of the registered entity.
	Instance EntityHandler

	failed         error
	eventSequence  int64
	shouldSnapshot bool
	events         []interface{}
	ctx            context.Context

	forward     *protocol.Forward
	sideEffects []*protocol.SideEffect
}

// Emit is called by a command handler.
func (c *Context) Emit(event interface{}) {
	if c.failed != nil {
		// we can't fail sooner but won't handle events after one failed anymore.
		return
	}
	if err := c.Instance.HandleEvent(c, event); err != nil {
		c.fail(err)
		return
	}
	c.events = append(c.events, event)
	c.eventSequence++
	c.shouldSnapshot = c.shouldSnapshot || c.eventSequence >= c.EventSourcedEntity.SnapshotEvery
}

func (c *Context) Effect(serviceName string, command string, payload *any.Any, synchronous bool) {
	c.sideEffects = append(c.sideEffects, &protocol.SideEffect{
		ServiceName: serviceName,
		CommandName: command,
		Payload:     payload,
		Synchronous: synchronous,
	})
}

func (c *Context) Forward(serviceName string, command string, payload *any.Any) {
	c.forward = &protocol.Forward{
		ServiceName: serviceName,
		CommandName: command,
		Payload:     payload,
	}
}

// StreamCtx returns the context.Context for the contexts' current running stream.
func (c *Context) StreamCtx() context.Context {
	return c.ctx
}

func (c *Context) fail(err error) {
	c.failed = err
}

func (c *Context) resetSnapshotEvery() {
	c.shouldSnapshot = false
	c.eventSequence = 0
}

// marshalEventsAny marshals and the clears events emitted through the context.
func (c *Context) marshalEventsAny() ([]*any.Any, error) {
	events := make([]*any.Any, len(c.events))
	for i, evt := range c.events {
		event, err := encoding.MarshalAny(evt)
		if err != nil {
			return nil, err
		}
		events[i] = event
	}
	c.events = make([]interface{}, 0)
	return events, nil
}
