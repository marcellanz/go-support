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
	"github.com/golang/protobuf/proto"
)

func TestPNCounter_Increment(t *testing.T) {
	c := PNCounter{}
	c.Increment(10)
	if v := c.Value(); v != 10 {
		t.Errorf("c.Value: %v; want: %v", v, 10)
	}
}

func TestPNCounter_ApplyState(t *testing.T) {
	c := PNCounter{}
	c.applyState(pnEncDecState(&protocol.PNCounterState{
		Value: 10,
	}))
	if v := c.Value(); v != 10 {
		t.Errorf("c.Value: %v; want: %v", v, 10)
	}
	c.applyState(pnEncDecState(&protocol.PNCounterState{
		Value: -7,
	}))
	if v := c.Value(); v != -7 {
		t.Errorf("c.Value: %v; want: %v", v, -7)
	}
}

func TestPNCounter_ApplyDelta(t *testing.T) {
	c := PNCounter{}
	c.applyDelta(pnEncDecDelta(&protocol.PNCounterDelta{
		Change: 10,
	}))
	if v := c.Value(); v != 10 {
		t.Errorf("c.Value: %v; want: %v", v, 10)
	}
	c.applyDelta(pnEncDecDelta(&protocol.PNCounterDelta{
		Change: 19,
	}))
	if v := c.Value(); v != 29 {
		t.Errorf("c.Value: %v; want: %v", v, 29)
	}
}

func TestPNCounter_Delta(t *testing.T) {
	c := PNCounter{}
	c.Increment(10)
	if d := pnEncDecDelta(c.Delta()).Change; d != 10 {
		t.Errorf("c.Delta: %v; want: %v", d, 10)
	}
	c.ResetDelta()
	if d := pnEncDecDelta(c.Delta()).Change; d != 0 {
		t.Errorf("c.Delta: %v; want: %v", d, 0)
	}
	c.Decrement(3)
	if v := c.Value(); v != 7 {
		t.Errorf("c.Value: %v; want: %v", v, 7)
	}
	c.Decrement(4)
	if v := c.Value(); v != 3 {
		t.Errorf("c.Value: %v; want: %v", v, 3)
	}
	if d := pnEncDecDelta(c.Delta()).Change; d != -7 {
		t.Errorf("c.Delta: %v; want: %v", d, -7)
	}
	c.ResetDelta()
	if d := pnEncDecDelta(c.Delta()).Change; d != 0 {
		t.Errorf("c.Delta: %v; want: %v", d, 0)
	}
}

func TestPNCounter_State(t *testing.T) {
	c := PNCounter{}
	c.Increment(10)
	if v := pnEncDecState(c.State()).Value; v != 10 {
		t.Errorf("c.Value: %v; want: %v", v, 10)
	}
}

func pnEncDecState(s *protocol.PNCounterState) *protocol.PNCounterState {
	marshal, err := proto.Marshal(s)
	if err != nil {
		// we panic for convenience
		panic(err)
	}
	out := &protocol.PNCounterState{}
	if err := proto.Unmarshal(marshal, out); err != nil {
		panic(err)
	}
	return out
}

func pnEncDecDelta(s *protocol.PNCounterDelta) *protocol.PNCounterDelta {
	marshal, err := proto.Marshal(s)
	if err != nil {
		// we panic for convenience
		panic(err)
	}
	out := &protocol.PNCounterDelta{}
	if err := proto.Unmarshal(marshal, out); err != nil {
		panic(err)
	}
	return out
}
