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
	"testing"

	"github.com/cloudstateio/go-support/cloudstate/protocol"
)

func TestGCounter(t *testing.T) {
	delta := func(incr uint64) *protocol.CrdtDelta {
		return &protocol.CrdtDelta{
			Delta: &protocol.CrdtDelta_Gcounter{
				Gcounter: &protocol.GCounterDelta{
					Increment: incr,
				},
			},
		}
	}
	state := func(val uint64) *protocol.CrdtState {
		return &protocol.CrdtState{
			State: &protocol.CrdtState_Gcounter{
				Gcounter: &protocol.GCounterState{
					Value: val,
				},
			},
		}
	}

	t.Run("should reflect a value update", func(t *testing.T) {
		c := GCounter{}
		c.applyState(encDecState(state(29)))
		if v := c.Value(); v != 29 {
			t.Errorf("c.Value: %v; want: %d", v, 29)
		}
		c.applyState(encDecState(state(92)))
		if v := c.Value(); v != 92 {
			t.Errorf("c.Value: %v; want: %d", v, 92)
		}
	})

	t.Run("should reflect a lwwRegisterDelta update", func(t *testing.T) {
		c := GCounter{}
		c.applyDelta(encDecDelta(delta(29)))

		if c.Value() != 29 {
			t.Fail()
		}
		c.applyDelta(encDecDelta(delta(92)))
		if c.Value() != 29+92 {
			t.Fail()
		}
	})

	t.Run("should generate deltas", func(t *testing.T) {
		c := GCounter{}
		c.Increment(10)
		if c.delta != 10 {
			t.Errorf("c.lwwRegisterDelta: %v; want: %v", c.delta, 10)
		}
		c.ResetDelta()
		if c.delta != 0 {
			t.Errorf("c.lwwRegisterDelta: %v; want: %v", c.delta, 0)
		}
		c.Increment(3)
		if c.Value() != 13 {
			t.Errorf("c.lwwRegisterDelta: %v; want: %v", c.delta, 13)
		}
		c.Increment(4)
		if c.Value() != 17 {
			t.Errorf("c.lwwRegisterDelta: %v; want: %v", c.delta, 17)
		}
		c.ResetDelta()
		if c.delta != 0 {
			t.Errorf("c.lwwRegisterDelta: %v; want: %v", c.delta, 0)
		}
	})

	t.Run("should return its value", func(t *testing.T) {
		c := GCounter{}
		c.Increment(29)
		if encDecState(c.State()).GetGcounter().GetValue() != 29 {
			t.Fail()
		}
	})
	t.Run("should return its lwwRegisterDelta", func(t *testing.T) {
		c := GCounter{}
		c.Increment(29)
		if encDecDelta(c.Delta()).GetGcounter().GetIncrement() != 29 {
			t.Fail()
		}
	})
	t.Run("ResetDelta", func(t *testing.T) {
		c := GCounter{}
		c.Increment(29)
		c.ResetDelta()
		if c.value != 29 || c.delta != 0 {
			t.Fail()
		}
	})
	t.Run("applyDelta", func(t *testing.T) {
		c := GCounter{}
		c.Increment(10)
		delta := encDecDelta(delta(29))
		c.applyDelta(delta)
		if c.Value() != 10+29 {
			t.Fail()
		}
	})
	t.Run("applyState", func(t *testing.T) {
		c := GCounter{}
		c.Increment(10)
		c.applyState(encDecState(state(29)))
		if c.Value() != 29 {
			t.Fail()
		}
	})
}
