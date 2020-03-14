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

import "github.com/cloudstateio/go-support/cloudstate/protocol"

type PNCounter struct {
	value int64
	delta int64
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

func (c *PNCounter) State() *protocol.PNCounterState {
	return &protocol.PNCounterState{
		Value: c.value,
	}
}

func (c *PNCounter) Delta() *protocol.PNCounterDelta {
	return &protocol.PNCounterDelta{
		Change: c.delta,
	}
}

func (c *PNCounter) ResetDelta() {
	c.delta = 0
}

func (c *PNCounter) applyState(state *protocol.PNCounterState) {
	c.value = state.Value
}

func (c *PNCounter) applyDelta(delta *protocol.PNCounterDelta) {
	c.value += delta.Change
}
