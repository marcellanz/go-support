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

	t.Run("should reflect a state update", func(t *testing.T) {
		c := GCounter{}
		if err := c.applyState(encDecState(state(29))); err != nil {
			t.Fatal(err)
		}
		if v := c.Value(); v != 29 {
			t.Errorf("c.Value: %v; want: %d", v, 29)
		}
		if err := c.applyState(encDecState(state(92))); err != nil {
			t.Fatal(err)
		}
		if v := c.Value(); v != 92 {
			t.Errorf("c.Value: %v; want: %d", v, 92)
		}
	})

	t.Run("should reflect a delta update", func(t *testing.T) {
		c := NewGCounter()
		if err := c.applyDelta(encDecDelta(delta(10))); err != nil {
			t.Fatal(err)
		}
		if v := c.Value(); v != 10 {
			t.Errorf("c.Value: %v; want: %d", v, 10)
		}
		if err := c.applyDelta(encDecDelta(delta(5))); err != nil {
			t.Fatal(err)
		}
		if v := c.Value(); v != 15 {
			t.Errorf("c.Value: %v; want: %d", v, 15)
		}
	})

	t.Run("should generate deltas", func(t *testing.T) {
		c := GCounter{}
		c.Increment(10)
		if c.Delta().GetGcounter().GetIncrement() != 10 {
			t.Errorf("counter increment: %v; want: %v", c.delta, 10)
		}
		c.resetDelta()
		if c.Delta().GetGcounter().GetIncrement() != 0 {
			t.Errorf("counter increment: %v; want: %v", c.delta, 0)
		}
		c.Increment(3)
		if c.Value() != 13 {
			t.Errorf("counter increment: %v; want: %v", c.delta, 13)
		}
		c.Increment(4)
		if c.Value() != 17 {
			t.Errorf("counter increment: %v; want: %v", c.delta, 17)
		}
		// TODO: c.Delta().GetGcounter().GetIncrement() is 0 even if Delta() returns nil
		if c.Delta().GetGcounter().GetIncrement() != 7 {
			t.Errorf("counter increment: %v; want: %v", c.delta, 7)
		}
		c.resetDelta()
		if d := c.Delta(); d != nil {
			t.Errorf("c.Delta() should be nil, but was not")
		}
	})

	t.Run("should return its state", func(t *testing.T) {
		c := GCounter{}
		c.Increment(10)
		if v := encDecState(c.State()).GetGcounter().GetValue(); v != 10 {
			t.Errorf("c.Value: %v; want: %d", v, 10)
		}
		c.resetDelta()
		if d := c.Delta(); d != nil {
			t.Errorf("c.Delta() should be nil, but was not")
		}
	})
}

func TestGCounterAdditional(t *testing.T) {
	t.Run("should report hasDelta", func(t *testing.T) {
		c := NewGCounter()
		if c.HasDelta() {
			t.Errorf("c.HasDelta() but should not")
		}
		c.Increment(29)
		if !c.HasDelta() {
			t.Errorf("c.HasDelta() is false, but should not")
		}
	})

	t.Run("should catch illegal delta applied", func(t *testing.T) {
		c := NewGCounter()
		err := c.applyDelta(&protocol.CrdtDelta{
			Delta: &protocol.CrdtDelta_Pncounter{
				Pncounter: &protocol.PNCounterDelta{
					Change: 11,
				},
			},
		})
		if err == nil {
			t.Errorf("c.applyDelta() has to err, but did not")
		}
	})

	t.Run("should catch illegal state applied", func(t *testing.T) {
		c := NewGCounter()
		err := c.applyState(&protocol.CrdtState{
			State: &protocol.CrdtState_Pncounter{
				Pncounter: &protocol.PNCounterState{
					Value: 11,
				},
			},
		})
		if err == nil {
			t.Errorf("c.applyState() has to err, but did not")
		}
	})
}
