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

	"github.com/cloudstateio/go-support/cloudstate/protocol"
)

type GCounter struct {
	value uint64
	delta uint64
}

var _ CRDT = (*GCounter)(nil)

func NewGCounter() *GCounter {
	return &GCounter{}
}

func (c *GCounter) Value() uint64 {
	return c.value
}

func (c *GCounter) Increment(i uint64) {
	c.value += i
	c.delta += i
}

func (c *GCounter) State() *protocol.CrdtState {
	return &protocol.CrdtState{
		State: &protocol.CrdtState_Gcounter{
			Gcounter: &protocol.GCounterState{
				Value: c.value,
			},
		},
	}
}

func (c GCounter) HasDelta() bool {
	return c.delta > 0
}

func (c *GCounter) Delta() *protocol.CrdtDelta {
	if c.delta == 0 {
		return nil
	}
	return &protocol.CrdtDelta{
		Delta: &protocol.CrdtDelta_Gcounter{
			Gcounter: &protocol.GCounterDelta{
				Increment: c.delta,
			},
		},
	}
}

func (c *GCounter) resetDelta() {
	c.delta = 0
}

func (c *GCounter) applyState(state *protocol.CrdtState) error {
	s := state.GetGcounter()
	if s == nil {
		return errors.New(fmt.Sprintf("unable to apply state %v to GCounter", state))
	}
	c.value = state.GetGcounter().GetValue()
	return nil
}

func (c *GCounter) applyDelta(delta *protocol.CrdtDelta) error {
	d := delta.GetGcounter()
	if d == nil {
		return errors.New(fmt.Sprintf("unable to apply delta %v to GCounter", delta))
	}
	c.value += d.GetIncrement()
	return nil
}
