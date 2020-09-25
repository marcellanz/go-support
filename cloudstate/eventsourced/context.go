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
	"github.com/golang/protobuf/ptypes/any"
)

type Context struct {
	EntityId EntityId
	// EventSourcedEntity describes the instance hold by the EntityInstance.
	EventSourcedEntity *Entity
	// Instance is an instance of the registered entity.
	Instance EntityHandler

	failed        error
	eventSequence int64
	events        []interface{}
	ctx           context.Context
}

// Emit is called by a command handler.
func (c *Context) Emit(event interface{}) {
	if c.failed != nil {
		// we can't fail sooner but won't handle events after one failed anymore.
		return
	}
	c.events = append(c.events, event)
	c.eventSequence++
}

// StreamCtx returns the context.Context for the contexts' current running stream.
func (c *Context) StreamCtx() context.Context {
	return c.ctx
}

func (c *Context) fail(err error) {
	c.failed = err
}

func (c *Context) shouldSnapshot() bool {
	return c.eventSequence >= c.EventSourcedEntity.SnapshotEvery
}

func (c *Context) resetSnapshotEvery() {
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
