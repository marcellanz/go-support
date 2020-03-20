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

	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/golang/protobuf/ptypes/any"
)

func TestGset(t *testing.T) {
	state := func(x []*any.Any) *protocol.CrdtState {
		return &protocol.CrdtState{
			State: &protocol.CrdtState_Gset{
				Gset: &protocol.GSetState{
					Items: x,
				},
			},
		}
	}
	delta := func(x []*any.Any) *protocol.CrdtDelta {
		return &protocol.CrdtDelta{
			Delta: &protocol.CrdtDelta_Gset{
				Gset: &protocol.GSetDelta{
					Added: x,
				},
			},
		}
	}

	t.Run("should have no elements when instantiated", func(t *testing.T) {
		s := NewGSet()
		if s.Size() != 0 {
			t.Fail()
		}
		if s.HasDelta() {
			t.Fatal("has delta but should not")
		}
		itemsLen := len(encDecState(s.State()).GetGset().GetItems())
		if itemsLen != 0 {
			t.Fatalf("len(items): %v; want: %v", itemsLen, 0)
		}
	})

	t.Run("should reflect a state update", func(t *testing.T) {
		s := NewGSet()
		if err := s.applyState(state(append(make([]*any.Any, 0),
			encoding.String("one"),
			encoding.String("two")),
		)); err != nil {
			t.Fatal(err)
		}
		if size := s.Size(); size != 2 {
			t.Fatalf("s.Size(): %v; want: %v", size, 2)
		}
		if !contains(s.Value(), "one", "two") {
			t.Fatalf("set values should contain 'one' and 'two', but was: %v", s.Value())
		}
		delta := s.Delta()
		if delta != nil {
			t.Fatalf("set delta should be nil but was not: %+v", delta)
		}
		s.ResetDelta()
		if slen := len(encDecState(s.State()).GetGset().GetItems()); slen != 2 {
			t.Fatalf("set state length: %v; want: %v", s.Size(), 2)
		}
	})

	t.Run("should generate an add delta", func(t *testing.T) {
		s := NewGSet()
		s.Add(encoding.String("one"))
		if !contains(s.Value(), "one") {
			t.Fatal("set should have a: one")
		}
		if s.Size() != 1 {
			t.Fatalf("s.Size(): %v; want: %v", s.Size(), 1)
		}
		delta := encDecDelta(s.Delta())
		s.ResetDelta()
		addedLen := len(delta.GetGset().GetAdded())
		if addedLen != 1 {
			t.Fatalf("s.Size(): %v; want: %v", addedLen, 1)
		}
		if !contains(delta.GetGset().GetAdded(), "one") {
			t.Fatalf("set should have a: one")
		}
		if s.HasDelta() {
			t.Fatalf("has but should not")
		}
		s.Add(encoding.String("two"))
		s.Add(encoding.String("three"))
		if s.Size() != 3 {
			t.Fatalf("s.Size(): %v; want: %v", s.Size(), 3)
		}
		delta2 := encDecDelta(s.Delta())
		addedLen2 := len(delta2.GetGset().GetAdded())
		if addedLen2 != 2 {
			t.Fatalf("s.Size(): %v; want: %v", addedLen2, 2)
		}
		if !contains(delta2.GetGset().GetAdded(), "two", "three") {
			t.Fatalf("delta should include two, three")
		}
		s.ResetDelta()
		if s.HasDelta() {
			t.Fatalf("has delta but should not")
		}
	})

	t.Run("should not generate a delta when an already existing element is added", func(t *testing.T) {
		s := NewGSet()
		s.Add(encoding.String("one"))
		s.ResetDelta()
		s.Add(encoding.String("one"))
		if s.Size() != 1 {
			t.Fatalf("s.Size(): %v; want: %v", s.Size(), 1)
		}
		if s.HasDelta() {
			t.Fatalf("has delta but should not")
		}
	})
	t.Run("should reflect a delta add", func(t *testing.T) {
		s := NewGSet()
		s.Add(encoding.String("one"))
		s.ResetDelta()
		s.applyDelta(delta(append(make([]*any.Any, 0), encoding.String("two"))))
		if s.Size() != 2 {
			t.Fatalf("s.Size(): %v; want: %v", s.Size(), 2)
		}
		if !contains(s.Value(), "one", "two") {
			t.Fatalf("delta should include two, three")
		}
		if s.HasDelta() {
			t.Fatalf("has delta but should not")
		}
		state := encDecState(s.State())

		if len(state.GetGset().GetItems()) != 2 {
			t.Fatalf("state.GetItems(): %v; want: %v", state.GetGset().GetItems(), 2)
		}
	})

	t.Run("should work with protobuf types", func(t *testing.T) {
		s := NewGSet()
		type Example struct {
			Field1 string
		}
		s.Add(encoding.Struct(&Example{Field1: "one"}))
		s.ResetDelta()
		s.Add(encoding.Struct(&Example{Field1: "one"}))
		if s.Size() != 1 {
			t.Fatalf("s.Size(): %v; want: %v", s.Size(), 1)
		}
		s.Add(encoding.Struct(&Example{Field1: "two"}))
		if s.Size() != 2 {
			t.Fatalf("s.Size(): %v; want: %v", s.Size(), 2)
		}
		delta := encDecDelta(s.Delta())
		if len(delta.GetGset().GetAdded()) != 1 {
			t.Fatalf("s.Size(): %v; want: %v", len(delta.GetGset().GetAdded()), 1)
		}
		foundOne := false
		for _, v := range delta.GetGset().GetAdded() {
			e := Example{}
			encoding.UnmarshalJSON(v, &e)
			if e.Field1 == "two" {
				foundOne = true
			}
		}
		if !foundOne {
			t.Errorf("delta should include two")
		}
	})

	type a struct {
		B string
		C int
	}

	t.Run("add primitive type", func(t *testing.T) {
		s := NewGSet()
		s.Add(encoding.Int32(5))
		for _, any := range s.value {
			p, err := encoding.UnmarshalPrimitive(any)
			if err != nil {
				t.Fail()
			}
			i, ok := p.(int32)
			if !ok {
				t.Fail()
			}
			if i != 5 {
				t.Fail()
			}
		}
	})

	t.Run("add struct stable", func(t *testing.T) {
		s := NewGSet()
		json, err := encoding.JSON(
			a{
				B: "hupps",
				C: 7,
			})
		if err != nil {
			t.Error(err)
		}
		s.Add(json)
		if s.Size() != 1 {
			t.Errorf("s.Size %v; want: %v", s.Size(), 1)
		}
		json, err = encoding.JSON(
			a{
				B: "hupps",
				C: 7,
			})
		if err != nil {
			t.Error(err)
		}
		s.Add(json)
		if s.Size() != 1 {
			t.Errorf("s.Size %v; want: %v", s.Size(), 1)
		}
	})
}

func TestGSetAdditional(t *testing.T) {
	t.Run("apply invalid delta", func(t *testing.T) {
		s := NewGSet()
		if err := s.applyDelta(&protocol.CrdtDelta{
			Delta: &protocol.CrdtDelta_Flag{
				Flag: &protocol.FlagDelta{
					Value: false,
				},
			},
		}); err == nil {
			t.Fatal("gset applyDelta should err but did not")
		}
	})
	t.Run("apply invalid state", func(t *testing.T) {
		s := NewGSet()
		if err := s.applyState(&protocol.CrdtState{
			State: &protocol.CrdtState_Flag{
				Flag: &protocol.FlagState{
					Value: false,
				},
			},
		}); err == nil {
			t.Fatal("gset applyState should err but did not")
		}
	})
}
