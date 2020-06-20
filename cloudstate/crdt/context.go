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
	"errors"
	"fmt"
)

type Context struct {
	EntityId EntityId
	// Entity describes the instance that is used as an entity
	Entity *Entity
	// Instance is the instance of the entity this context is for.
	Instance interface{}
	// the root crdt managed by this user function
	crdt CRDT

	streamedCtx map[CommandId]*CommandContext
	created     bool
	active      bool
	deleted     bool
	failed      error
	ctx         context.Context
}

func (c *Context) SetCRDT(newCRDT CRDT) error {
	if c.crdt != nil {
		return fmt.Errorf("crdt has been already created")
	}
	c.crdt = newCRDT
	c.created = true
	return nil
}

func (c *Context) CRDT() CRDT {
	return c.crdt
}

// Fail fails the command with the given message.
func (c *Context) Fail(err error) {
	// TODO: has to be active, has to be not yet failed
	// "fail(…) already previously invoked!"
	c.failed = fmt.Errorf("failed with %v: %w", err, ErrFailCalled)
}

func (c *Context) Delete() {
	c.deleted = true
	c.crdt = nil
}

// initDefault initializes the CRDT with a default value if not already set.
func (c *Context) initDefault() error {
	if c.crdt != nil {
		c.Entity.SetFunc(c, c.crdt)
		return nil
	}
	if c.Entity.DefaultFunc == nil {
		return errors.New("no Entity.DefaultFunc defined")
	}
	c.crdt = c.Entity.DefaultFunc(c)
	if c.failed != nil {
		return c.failed
	}
	if c.crdt == nil {
		return errors.New("no default CRDT set by Entity.DefaultFunc")
	}
	c.Entity.SetFunc(c, c.crdt)
	c.created = true
	return nil
}

func (c *Context) deactivate() {
	c.active = false
}
