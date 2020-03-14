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

func TestORSet(t *testing.T) {
	t.Run("should reflect a state update", func(t *testing.T) {
		s := NewORSet()
		s.ApplyState(&protocol.ORSetState{
			Items: append(make([]*any.Any, 0), encoding.String("one"), encoding.String("two")),
		})
		if s.Size() != 2 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 2)
		}
	})

	t.Run("should generate an add lwwRegisterDelta", func(t *testing.T) {
		s := NewORSet()
		s.Add(encoding.String("one"))
		if s.Size() != 1 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 1)
		}
		delta := orSetEncDecDelta(s.Delta())
		s.ResetDelta()
		if len(delta.GetAdded()) != 1 {
			t.Errorf("s.Delta()).GetAdded()): %v; want: %v", len(orSetEncDecDelta(s.Delta()).GetAdded()), 1)
		}
		if !contains(delta.Added, "one") {
			t.Error("did not found one")
		}
		if s.HasDelta() {
			t.Errorf("set has lwwRegisterDelta")
		}
		s.Add(encoding.String("two"))
		s.Add(encoding.String("three"))
		if s.Size() != 3 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 3)
		}
		addedLen := len(orSetEncDecDelta(s.Delta()).GetAdded())
		if addedLen != 2 {
			t.Errorf("len(GetAdded()): %v; want: %v", addedLen, 2)
		}
		if !contains(s.Added(), "two", "three") {
			t.Error("did not found two and three")
		}
		s.ResetDelta()
		if s.HasDelta() {
			t.Errorf("set has lwwRegisterDelta")
		}
	})

	t.Run("should generate a remove lwwRegisterDelta", func(t *testing.T) {
		s := NewORSet()
		s.Add(encoding.String("one"))
		s.Add(encoding.String("two"))
		s.Add(encoding.String("three"))
		s.ResetDelta()
		if !contains(s.Value(), "one", "two", "three") {
			t.Errorf("removed does not include: one, two, three")
		}
		s.Remove(encoding.String("one"))
		s.Remove(encoding.String("two"))
		if s.Size() != 1 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 1)
		}
		if contains(s.Value(), "one") {
			t.Errorf("set should not include one")
		}
		if contains(s.Value(), "two") {
			t.Errorf("set should not include two")
		}
		if !contains(s.Value(), "three") {
			t.Errorf("set should include three")
		}
		delta := orSetEncDecDelta(s.Delta())
		if len(delta.GetRemoved()) != 2 {
			t.Errorf("len(lwwRegisterDelta.GetRemoved()): %v; want: %v", len(delta.GetRemoved()), 2)
		}
		if !contains(s.Removed(), "one", "two") {
			t.Errorf("removed does not include: one, two")
		}
	})

	t.Run("should generate a clear lwwRegisterDelta", func(t *testing.T) {
		s := NewORSet()
		s.Add(encoding.String("one"))
		s.Add(encoding.String("two"))
		_ = s.Delta()
		s.ResetDelta()
		s.Clear()
		if s.Size() != 0 {
			t.Errorf("s.Size(): %v; want: %v", len(s.Removed()), 0)
		}
		delta := orSetEncDecDelta(s.Delta())
		s.ResetDelta()
		if !delta.Cleared {
			t.Fail()
		}
		delta = s.Delta()
		if s.HasDelta() {
			t.Errorf("set has lwwRegisterDelta")
		}
	})

	t.Run("should generate a clear lwwRegisterDelta when everything is removed", func(t *testing.T) {
		s := NewORSet()
		s.Add(encoding.String("one"))
		s.Add(encoding.String("two"))
		s.ResetDelta()
		s.Remove(encoding.String("one"))
		s.Remove(encoding.String("two"))
		if s.Size() != 0 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 0)
		}
		delta := orSetEncDecDelta(s.Delta())
		s.ResetDelta()
		if !delta.Cleared {
			t.Errorf("lwwRegisterDelta.Cleared: %v; want: %v", delta.Cleared, true)
		}
		if s.HasDelta() {
			t.Errorf("set has lwwRegisterDelta")
		}
	})

	t.Run("should not generate a lwwRegisterDelta when an added element is removed", func(t *testing.T) {
		s := NewORSet()
		s.Add(encoding.String("one"))
		s.Add(encoding.String("two"))
		s.Delta()
		s.ResetDelta()
		s.Add(encoding.String("two"))
		s.Remove(encoding.String("two"))
		if s.Size() != 1 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 1)
		}
		//lwwRegisterDelta := orSetEncDecDelta(s.Delta())
		s.ResetDelta()
		if s.HasDelta() {
			t.Errorf("set has lwwRegisterDelta")
		}
	})

	t.Run("should not generate a lwwRegisterDelta when a removed element is added", func(t *testing.T) {
		s := NewORSet()
		s.Add(encoding.String("one"))
		s.Add(encoding.String("two"))
		s.Delta()
		s.ResetDelta()
		s.Remove(encoding.String("two"))
		s.Add(encoding.String("two"))
		if s.Size() != 2 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 2)
		}
		if s.HasDelta() {
			t.Errorf("set has lwwRegisterDelta")
		}
	})

	t.Run("should not generate a lwwRegisterDelta when an already existing element is added", func(t *testing.T) {
		s := NewORSet()
		s.Add(encoding.String("one"))
		s.ResetDelta()
		s.Add(encoding.String("one"))
		if s.Size() != 1 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 1)
		}
		if s.HasDelta() {
			t.Errorf("set has lwwRegisterDelta")
		}
	})
	t.Run("should not generate a lwwRegisterDelta when a non existing element is removed", func(t *testing.T) {
		s := NewORSet()
		s.Add(encoding.String("one"))
		s.ResetDelta()
		s.Remove(encoding.String("two"))
		if s.Size() != 1 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 1)
		}
		if s.HasDelta() {
			t.Errorf("set has lwwRegisterDelta")
		}
	})
	t.Run("clear all other deltas when the set is cleared", func(t *testing.T) {
		s := NewORSet()
		s.Add(encoding.String("one"))
		s.ResetDelta()
		s.Add(encoding.String("two"))
		s.Remove(encoding.String("one"))
		s.Clear()
		if s.Size() != 0 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 0)
		}
		delta := orSetEncDecDelta(s.Delta())
		if !delta.Cleared {
			t.Errorf("lwwRegisterDelta.Cleared: %v; want: %v", delta.Cleared, true)
		}
		if len(delta.GetAdded()) != 0 {
			t.Errorf("len(lwwRegisterDelta.GetAdded()): %v; want: %v", len(delta.GetAdded()), true)
		}
		if len(delta.GetRemoved()) != 0 {
			t.Errorf("len(lwwRegisterDelta.GetRemoved): %v; want: %v", len(delta.GetRemoved()), true)
		}
	})
	t.Run("should reflect a lwwRegisterDelta add", func(t *testing.T) {
		s := NewORSet()
		s.Add(encoding.String("one"))
		s.ResetDelta()
		s.ApplyDelta(orSetEncDecDelta(&protocol.ORSetDelta{
			Added: append(make([]*any.Any, 0), encoding.String("two")),
		}))
		if s.Size() != 2 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 2)
		}
		s.ResetDelta()
		if s.HasDelta() {
			t.Errorf("set has lwwRegisterDelta")
		}
		stateLen := len(orSetEncDecState(s.State()).GetItems())
		if stateLen != 2 {
			t.Errorf("len(GetItems()): %v; want: %v", stateLen, 2)
		}
	})

	t.Run("should reflect a lwwRegisterDelta remove", func(t *testing.T) {
		s := NewORSet()
		s.Add(encoding.String("one"))
		s.Add(encoding.String("two"))
		s.ApplyDelta(orSetEncDecDelta(&protocol.ORSetDelta{
			Removed: append(make([]*any.Any, 0), encoding.String("two")),
		}))
		if s.Size() != 1 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 1)
		}
		stateLen := len(orSetEncDecState(s.State()).GetItems())
		if stateLen != 1 {
			t.Errorf("len(GetItems()): %v; want: %v", stateLen, 1)
		}
	})

	t.Run("should reflect a lwwRegisterDelta clear", func(t *testing.T) {
		s := NewORSet()
		s.Add(encoding.String("one"))
		s.Add(encoding.String("two"))
		s.ResetDelta()
		s.ApplyDelta(orSetEncDecDelta(&protocol.ORSetDelta{
			Cleared: true,
		}))
		if s.Size() != 0 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 0)
		}
		stateLen := len(orSetEncDecState(s.State()).GetItems())
		if stateLen != 0 {
			t.Errorf("len(GetItems()): %v; want: %v", stateLen, 0)
		}
	})

	t.Run("should work with protobuf types", func(t *testing.T) {
		s := NewORSet()
		type Example struct {
			Field1 string
		}
		one := encoding.Struct(Example{Field1: "one"})
		s.Add(one)
		two := encoding.Struct(Example{Field1: "two"})
		s.Add(two)
		s.ResetDelta()
		s.Remove(encoding.Struct(Example{Field1: "one"}))
		if s.Size() != 1 {
			t.Errorf("s.Size(): %v; want: %v", s.Size(), 1)
		}
		delta := orSetEncDecDelta(s.Delta())
		removedLen := len(delta.GetRemoved())
		if removedLen != 1 {
			t.Errorf("removedLen: %v; want: %v", removedLen, 1)
		}
		e := &Example{}
		err := encoding.UnmarshalJSON(delta.GetRemoved()[0], e)
		if err != nil || e.Field1 != "one" {
			t.Fail()
		}
	})
}

func TestRemoved(t *testing.T) {
	s := NewORSet()
	s.Add(encoding.String("one"))
	if len(s.Removed()) != 0 {
		t.Fail()
	}
	s.Delta()
	s.ResetDelta()
	s.Remove(encoding.String("one"))
	if len(s.Removed()) != 0 {
		t.Fail()
	}
}

func orSetEncDecState(s *protocol.ORSetState) *protocol.ORSetState {
	marshal, err := proto.Marshal(s)
	if err != nil {
		// we panic for convenience
		panic(err)
	}
	out := &protocol.ORSetState{}
	if err := proto.Unmarshal(marshal, out); err != nil {
		panic(err)
	}
	return out
}

func orSetEncDecDelta(s *protocol.ORSetDelta) *protocol.ORSetDelta {
	marshal, err := proto.Marshal(s)
	if err != nil {
		// we panic for convenience
		panic(err)
	}
	out := &protocol.ORSetDelta{}
	if err := proto.Unmarshal(marshal, out); err != nil {
		panic(err)
	}
	return out
}
