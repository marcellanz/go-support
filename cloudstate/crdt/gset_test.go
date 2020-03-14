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
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
)

func TestGset(t *testing.T) {
	t.Run("should have no elements when instantiated", func(t *testing.T) {
		s := NewGSet()
		if s.Size() != 0 {
			t.Fail()
		}
		if s.HasDelta() {
			t.Errorf("has lwwRegisterDelta but should not")
		}
		itemsLen := len(gSetEncDecState(s.State()).Items)
		if itemsLen != 0 {
			t.Errorf("len(items): %v; want: %v", itemsLen, 0)
		}
	})
	t.Run("should generate an add lwwRegisterDelta", func(t *testing.T) {
		s := NewGSet()
		s.Add(encoding.String("one"))
		if !contains(s.Value(), "one") {
			t.Errorf("set should have a: one")
		}
		if s.Size() != 1 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 1)
		}
		delta := gSetEncDecDelta(s.Delta())
		s.ResetDelta()
		addedLen := len(delta.GetAdded())
		if addedLen != 1 {
			t.Errorf("s.Size(): %v; want: %v", addedLen, 1)
		}
		if !contains(delta.GetAdded(), "one") {
			t.Errorf("set should have a: one")
		}
		if s.HasDelta() {
			t.Errorf("has lwwRegisterDelta but should not")
		}
		s.Add(encoding.String("two"))
		s.Add(encoding.String("three"))
		if s.Size() != 3 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 3)
		}
		delta2 := gSetEncDecDelta(s.Delta())
		addedLen2 := len(delta2.GetAdded())
		if addedLen2 != 2 {
			t.Errorf("s.Size(): %v; want: %v", addedLen2, 2)
		}
		if !contains(delta2.GetAdded(), "two", "three") {
			t.Errorf("lwwRegisterDelta should include two, three")
		}
		s.ResetDelta()
		if s.HasDelta() {
			t.Errorf("has lwwRegisterDelta but should not")
		}
	})

	t.Run("should not generate a lwwRegisterDelta when an already existing element is added", func(t *testing.T) {
		s := NewGSet()
		s.Add(encoding.String("one"))
		s.ResetDelta()
		s.Add(encoding.String("one"))
		if s.Size() != 1 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 1)
		}
		if s.HasDelta() {
			t.Errorf("has lwwRegisterDelta but should not")
		}
	})
	t.Run("should reflect a lwwRegisterDelta add", func(t *testing.T) {
		s := NewGSet()
		s.Add(encoding.String("one"))
		s.ResetDelta()
		s.ApplyDelta(&protocol.GSetDelta{
			Added: append(make([]*any.Any, 0), encoding.String("two")),
		})
		if s.Size() != 2 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 2)
		}
		if !contains(s.Value(), "one", "two") {
			t.Errorf("lwwRegisterDelta should include two, three")
		}
		if s.HasDelta() {
			t.Errorf("has lwwRegisterDelta but should not")
		}
		state := gSetEncDecState(s.State())

		if len(state.GetItems()) != 2 {
			t.Errorf("state.GetItems(): %v; want: %v", state.GetItems(), 2)
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
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 1)
		}
		s.Add(encoding.Struct(&Example{Field1: "two"}))
		if s.Size() != 2 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 2)
		}
		delta := gSetEncDecDelta(s.Delta())
		if len(delta.GetAdded()) != 1 {
			t.Errorf("s.Size(): %v; want: %v", len(delta.GetAdded()), 1)
		}
		foundOne := false
		for _, v := range delta.GetAdded() {
			e := Example{}
			encoding.UnmarshalJSON(v, &e)
			if e.Field1 == "two" {
				foundOne = true
			}
		}
		if !foundOne {
			t.Errorf("lwwRegisterDelta should include two")
		}
	})
	/*
	  it("should work with protobuf types", () => {

	    const lwwRegisterDelta = roundTripDelta(set.getAndResetDelta());
	    lwwRegisterDelta.gset.added.should.have.lengthOf(1);
	    fromAnys(lwwRegisterDelta.gset.added)[0].field1.should.equal("two");
	  });
	*/

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

func gSetEncDecState(s *protocol.GSetState) *protocol.GSetState {
	marshal, err := proto.Marshal(s)
	if err != nil {
		// we panic for convenience
		panic(err)
	}
	out := &protocol.GSetState{}
	if err := proto.Unmarshal(marshal, out); err != nil {
		panic(err)
	}
	return out
}

func gSetEncDecDelta(s *protocol.GSetDelta) *protocol.GSetDelta {
	marshal, err := proto.Marshal(s)
	if err != nil {
		// we panic for convenience
		panic(err)
	}
	out := &protocol.GSetDelta{}
	if err := proto.Unmarshal(marshal, out); err != nil {
		panic(err)
	}
	return out
}
