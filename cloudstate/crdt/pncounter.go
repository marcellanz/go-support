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

type PNCounter struct {
	value int64
	delta int64
}

var _ CRDT = (*PNCounter)(nil)

func NewPNCounter() *PNCounter {
	return &PNCounter{}
}

func (c *PNCounter) Value() int64 {
	return c.value
}

func (c *PNCounter) Increment(i int64) {
	c.value += i
	c.delta += i
}

func (c *PNCounter) Decrement(i int64) {
	c.value -= i
	c.delta -= i
}

func (c *PNCounter) State() *protocol.CrdtState {
	return &protocol.CrdtState{
		State: &protocol.CrdtState_Pncounter{
			Pncounter: &protocol.PNCounterState{
				Value: c.value,
			},
		},
	}
}

func (c *PNCounter) HasDelta() bool {
	return c.delta != 0
}

func (c *PNCounter) Delta() *protocol.CrdtDelta {
	if c.delta == 0 {
		return nil
	}
	return &protocol.CrdtDelta{
		Delta: &protocol.CrdtDelta_Pncounter{
			Pncounter: &protocol.PNCounterDelta{
				Change: c.delta,
			},
		},
	}
}

func (c *PNCounter) resetDelta() {
	c.delta = 0
}

func (c *PNCounter) applyState(state *protocol.CrdtState) error {
	s := state.GetPncounter()
	if s == nil {
		return errors.New(fmt.Sprintf("unable to apply state %v to PNCounter", state))
	}
	c.value = s.GetValue()
	return nil
}

func (c *PNCounter) applyDelta(delta *protocol.CrdtDelta) error {
	d := delta.GetPncounter()
	if d == nil {
		return errors.New(fmt.Sprintf("unable to apply delta %v to PNCounter", delta))
	}
	c.value += d.GetChange()
	return nil
}
