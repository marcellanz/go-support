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
	"context"
	"fmt"
)

type Context struct {
	EntityId EntityId
	// Entity describes the instance that is used as an entity
	Entity *Entity
	// Instance is an instance of the Entity.Entity
	Instance interface{}

	// the root crdt managed by this user function
	crdt CRDT

	// active indicates if this context is active.
	created bool // TODO: inactivate a context in case of errors
	active  bool // TODO: inactivate a context in case of errors
	deleted bool // TODO: clientAction might check if the entity is deleted
	// ctx is the stream context
	ctx         context.Context
	failed      error
	streamedCtx map[CommandId]*StreamContext
}

func (c *Context) CRDT() CRDT {
	return c.crdt
}

func (c *Context) SetCRDT(newCRDT CRDT) error {
	if c.crdt != nil {
		return fmt.Errorf("crdt has been already created")
	}
	c.crdt = newCRDT
	c.created = true
	return nil
}

// initDefault initializes the crdt with a default value if not already set.
func (c *Context) initDefault() {
	if c.crdt != nil || c.Entity.DefaultFunc == nil {
		return
	}
	c.crdt = c.Entity.DefaultFunc(c)
	c.created = true
}

func (c *Context) deactivate() {
	c.active = false
}

func (c *Context) delete() {
	c.deleted = true
	c.crdt = nil
}
