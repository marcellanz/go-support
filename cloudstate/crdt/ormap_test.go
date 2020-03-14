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
	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/golang/protobuf/ptypes/any"
	"testing"
)

func TestORMap(t *testing.T) {
	t.Run("should have no elements when instantiated", func(t *testing.T) {
		m := NewORMap()
		if m.Size() != 0 {
			t.Fatalf("m.Size(): %v; want: %v", m.Size(), 0)
		}
		if m.Delta() != nil {
			t.Fatal("m.Delta() is not nil but should")
		}
		m.ResetDelta()
		if elen := len(encDecState(m.State()).GetOrmap().Entries); elen > 0 {
			t.Fatalf("len(Entries): %v; want: %v", elen, 0)
		}
	})
	t.Run("should reflect a state update", func(t *testing.T) {
		m := NewORMap()
		if err := m.applyState(encDecState(
			&protocol.CrdtState{
				State: &protocol.CrdtState_Ormap{
					Ormap: &protocol.ORMapState{
						Entries: append(make([]*protocol.ORMapEntry, 0),
							&protocol.ORMapEntry{
								Key: encoding.String("one"),
								Value: (&GCounter{
									value: 5,
								}).State(),
							},
							&protocol.ORMapEntry{
								Key: encoding.String("two"),
								Value: (&GCounter{
									value: 7,
								}).State(),
							},
						),
					},
				},
			},
		)); err != nil {
			t.Fatal(err)
		}
		if m.Size() != 2 {
			t.Fatalf("m.Size(): %v; want: %v", m.Size(), 2)
		}
		if v0 := m.Get(encoding.String("one")).GetGcounter().Value; v0 != 5 {
			t.Fatalf("GCounter.Value: %v; want: %v", v0, 5)
		}
		if v0 := m.Get(encoding.String("two")).GetGcounter().Value; v0 != 7 {
			t.Fatalf("GCounter.Value: %v; want: %v", v0, 7)
		}
		if d := m.Delta(); d != nil {
			t.Fatalf("m.Delta(): %v; want: %v", d, nil)
		}
		m.ResetDelta()
		if l := len(encDecState(m.State()).GetOrmap().Entries); l != 2 {
			t.Fatalf("len(map.Entries): %v; want: %v", l, 2)
		}
	})
	t.Run("should generate an add delta", func(t *testing.T) {
		m := NewORMap()
		m.SetGCounter(encoding.String("one"), NewGCounter())
		if !m.HasKey(encoding.String("one")) {
			t.Fatal("m has no 'one' key")
		}
		if s := m.Size(); s != 1 {
			t.Fatalf("m.Size(): %v; want: %v", s, 1)
		}
		delta := m.Delta()
		m.ResetDelta()
		if l := len(encDecDelta(delta).GetOrmap().GetAdded()); l != 1 {
			t.Fatalf("delta added length: %v; want: %v", l, 1)
		}
		entry := delta.GetOrmap().GetAdded()[0]
		if k := encoding.DecodeString(entry.GetKey()); k != "one" {
			t.Fatalf("key: %v; want: %v", k, "one")
		}
		if v := entry.GetValue().GetGcounter().GetValue(); v != 0 {
			t.Fatalf("GCounter.Value: %v; want: %v", v, 0)
		}
		if d := m.Delta(); d != nil {
			t.Fatalf("m.Delta(): %v; want: %v", d, nil)
		}

		m.SetGCounter(encoding.String("two"), NewGCounter())
		counter, err := m.GCounter(encoding.String("two"))
		if err != nil {
			t.Fatal(err)
		}
		counter.Increment(10)
		if s := m.Size(); s != 2 {
			t.Fatalf("m.Size(): %v; want: %v", s, 2)
		}
		delta2 := encDecDelta(m.Delta())
		m.ResetDelta()
		if l := len(delta2.GetOrmap().GetAdded()); l != 1 {
			t.Fatalf("delta added length: %v; want: %v", l, 1)
		}
		entry2 := delta2.GetOrmap().GetAdded()[0]
		if k := encoding.DecodeString(entry2.GetKey()); k != "two" {
			t.Fatalf("key: %v; want: %v", k, "two")
		}
		if v := entry2.GetValue().GetGcounter().GetValue(); v != 10 {
			t.Fatalf("GCounter.Value: %v; want: %v", v, 10)
		}
		if d := m.Delta(); d != nil {
			t.Fatalf("m.Delta(): %v; want: %v", d, nil)
		}

		if l := len(delta2.GetOrmap().GetUpdated()); l != 0 {
			t.Fatalf("length of delta: %v; want: %v", l, 0)
		}
	})

	t.Run("should generate a remove delta", func(t *testing.T) {
		m := NewORMap()
		m.SetGCounter(encoding.String("one"), NewGCounter())
		m.SetGCounter(encoding.String("two"), NewGCounter())
		m.SetGCounter(encoding.String("three"), NewGCounter())
		m.Delta()
		m.ResetDelta()
		if !m.HasKey(encoding.String("one")) {
			t.Fatalf("map should have key: %v but had not", "one")
		}
		if !m.HasKey(encoding.String("two")) {
			t.Fatalf("map should have key: %v but had not", "two")
		}
		if !m.HasKey(encoding.String("three")) {
			t.Fatalf("map should have key: %v but had not", "three")
		}
		if s := m.Size(); s != 3 {
			t.Fatalf("m.Size(): %v; want: %v", s, 3)
		}
		m.Delete(encoding.String("one"))
		m.Delete(encoding.String("two"))
		if s := m.Size(); s != 1 {
			t.Fatalf("m.Size(): %v; want: %v", s, 1)
		}
		if m.HasKey(encoding.String("one")) {
			t.Fatalf("map should not have key: %v but had", "one")
		}
		if m.HasKey(encoding.String("two")) {
			t.Fatalf("map should not have key: %v but had", "two")
		}
		if !m.HasKey(encoding.String("three")) {
			t.Fatalf("map should have key: %v but had not", "three")
		}
		delta := encDecDelta(m.Delta())
		m.ResetDelta()
		if l := len(delta.GetOrmap().GetRemoved()); l != 2 {
			t.Fatalf("length of delta.removed: %v; want: %v", l, 2)
		}
		if !contains(delta.GetOrmap().GetRemoved(), "one", "two") {
			t.Fatalf("delta.removed should contain keys 'one','two' but did not: %v", delta.GetOrmap().GetRemoved())
		}
		if d := m.Delta(); d != nil {
			t.Fatalf("m.Delta(): %v; want: %v", d, nil)
		}
	})
	t.Run("should generate an update delta", func(t *testing.T) {
		m := NewORMap()
		m.SetGCounter(encoding.String("one"), NewGCounter())
		m.ResetDelta()
		counter, err := m.GCounter(encoding.String("one"))
		if err != nil {
			t.Fatal(err)
		}
		counter.Increment(5)
		delta := encDecDelta(m.Delta())
		m.ResetDelta()
		if l := len(delta.GetOrmap().GetUpdated()); l != 1 {
			t.Fatalf("length of delta.updated: %v; want: %v", l, 1)
		}
		entry := delta.GetOrmap().GetUpdated()[0]
		if k := encoding.DecodeString(entry.GetKey()); k != "one" {
			t.Fatalf("key of updated entry was: %v; want: %v", k, "one")
		}
		if i := entry.GetDelta().GetGcounter().GetIncrement(); i != 5 {
			t.Fatalf("increment: %v; want: %v", i, 5)
		}
		if d := m.Delta(); d != nil {
			t.Fatalf("m.Delta(): %v; want: %v", d, nil)
		}

	})
	t.Run("should generate a clear delta", func(t *testing.T) {
		m := NewORMap()
		m.SetGCounter(encoding.String("one"), NewGCounter())
		m.SetGCounter(encoding.String("two"), NewGCounter())
		m.ResetDelta()
		m.Clear()
		if s := m.Size(); s != 0 {
			t.Fatalf("m.Size(): %v; want: %v", s, 0)
		}
		delta := encDecDelta(m.Delta())
		m.ResetDelta()
		if c := delta.GetOrmap().GetCleared(); !c {
			t.Fatalf("delta cleared: %v; want: %v", c, true)
		}
		if d := m.Delta(); d != nil {
			t.Fatalf("m.Delta(): %v; want: %v", d, nil)
		}
	})
	t.Run("should generate a clear delta when everything is removed", func(t *testing.T) {
		m := NewORMap()
		m.SetGCounter(encoding.String("one"), NewGCounter())
		m.SetGCounter(encoding.String("two"), NewGCounter())
		m.ResetDelta()
		m.Delete(encoding.String("one"))
		m.Delete(encoding.String("two"))
		if s := m.Size(); s != 0 {
			t.Fatalf("m.Size(): %v; want: %v", s, 0)
		}
		delta := encDecDelta(m.Delta())
		m.ResetDelta()
		if c := delta.GetOrmap().GetCleared(); !c {
			t.Fatalf("delta cleared: %v; want: %v", c, true)
		}
		if d := m.Delta(); d != nil {
			t.Fatalf("m.Delta(): %v; want: %v", d, nil)
		}
	})
	t.Run("should not generate a delta when an added element is removed", func(t *testing.T) {
		m := NewORMap()
		m.SetGCounter(encoding.String("one"), NewGCounter())
		m.ResetDelta()
		m.SetGCounter(encoding.String("two"), NewGCounter())
		m.Delete(encoding.String("two"))
		if s := m.Size(); s != 1 {
			t.Fatalf("m.Size(): %v; want: %v", s, 1)
		}
		if d := m.Delta(); d != nil {
			t.Fatalf("m.Delta(): %v; want: %v", d, nil)
		}
	})
	t.Run("should generate a delta when a removed element is added", func(t *testing.T) {
		m := NewORMap()
		m.SetGCounter(encoding.String("one"), NewGCounter())
		m.SetGCounter(encoding.String("two"), NewGCounter())
		m.ResetDelta()
		m.Delete(encoding.String("two"))
		m.SetGCounter(encoding.String("two"), NewGCounter())
		if s := m.Size(); s != 2 {
			t.Fatalf("m.Size(): %v; want: %v", s, 2)
		}
		delta := encDecDelta(m.Delta())
		m.ResetDelta()
		if l := len(delta.GetOrmap().GetRemoved()); l != 1 {
			t.Fatalf("length of delta.removed: %v; want: %v", l, 1)
		}
		if l := len(delta.GetOrmap().GetAdded()); l != 1 {
			t.Fatalf("length of delta.added: %v; want: %v", l, 1)
		}
		if l := len(delta.GetOrmap().GetUpdated()); l != 0 {
			t.Fatalf("length of delta.updated: %v; want: %v", l, 0)
		}
	})
	t.Run("should not generate a delta when a non existing element is removed", func(t *testing.T) {
		m := NewORMap()
		m.SetGCounter(encoding.String("one"), NewGCounter())
		m.ResetDelta()
		m.Delete(encoding.String("two"))
		if s := m.Size(); s != 1 {
			t.Fatalf("m.Size(): %v; want: %v", s, 1)
		}
		if d := m.Delta(); d != nil {
			t.Fatalf("m.Delta(): %v; want: %v", d, nil)
		}
	})
	t.Run("should generate a delta when an already existing element is set", func(t *testing.T) {
		m := NewORMap()
		m.SetGCounter(encoding.String("one"), NewGCounter())
		m.ResetDelta()
		m.SetGCounter(encoding.String("one"), NewGCounter())
		if s := m.Size(); s != 1 {
			t.Fatalf("m.Size(): %v; want: %v", s, 1)
		}
		delta := encDecDelta(m.Delta())
		m.ResetDelta()
		if l := len(delta.GetOrmap().GetRemoved()); l != 1 {
			t.Fatalf("length of delta.removed: %v; want: %v", l, 1)
		}
		if l := len(delta.GetOrmap().GetAdded()); l != 1 {
			t.Fatalf("length of delta.added: %v; want: %v", l, 1)
		}
		if l := len(delta.GetOrmap().GetUpdated()); l != 0 {
			t.Fatalf("length of delta.updated: %v; want: %v", l, 0)
		}
	})
	t.Run("clear all other deltas when the set is cleared", func(t *testing.T) {
		m := NewORMap()
		m.SetGCounter(encoding.String("one"), NewGCounter())
		m.SetGCounter(encoding.String("two"), NewGCounter())
		m.ResetDelta()
		counter, err := m.GCounter(encoding.String("two"))
		if err != nil {
			t.Fatal(err)
		}
		counter.Increment(10)
		m.SetGCounter(encoding.String("one"), NewGCounter())
		if s := m.Size(); s != 2 {
			t.Fatalf("m.Size(): %v; want: %v", s, 2)
		}
		m.Clear()
		if s := m.Size(); s != 0 {
			t.Fatalf("m.Size(): %v; want: %v", s, 0)
		}
		delta := encDecDelta(m.Delta())
		m.ResetDelta()
		if c := delta.GetOrmap().GetCleared(); !c {
			t.Fatalf("ormap cleared: %v; want: %v", c, true)
		}
		if l := len(delta.GetOrmap().GetAdded()); l != 0 {
			t.Fatalf("added len: %v; want: %v", l, 0)
		}
		if l := len(delta.GetOrmap().GetRemoved()); l != 0 {
			t.Fatalf("added len: %v; want: %v", l, 0)
		}
		if l := len(delta.GetOrmap().GetUpdated()); l != 0 {
			t.Fatalf("added len: %v; want: %v", l, 0)
		}
	})
	t.Run("should reflect a delta add", func(t *testing.T) {
		m := NewORMap()
		m.SetGCounter(encoding.String("one"), NewGCounter())
		m.ResetDelta()
		err := m.applyDelta(encDecDelta(&protocol.CrdtDelta{
			Delta: &protocol.CrdtDelta_Ormap{Ormap: &protocol.ORMapDelta{
				Added: append(make([]*protocol.ORMapEntry, 0), &protocol.ORMapEntry{
					Key: encoding.String("two"),
					Value: &protocol.CrdtState{
						State: &protocol.CrdtState_Gcounter{
							Gcounter: &protocol.GCounterState{
								Value: 4,
							},
						},
					},
				}),
			}},
		}))
		if err != nil {
			t.Fatal(err)
		}
		if s := m.Size(); s != 2 {
			t.Fatalf("m.Size(): %v; want: %v", s, 2)
		}
		if !contains(m.Keys(), "one", "two") {
			t.Fatalf("m.Keys() should include 'one','two' but did not: %v", m.Keys())
		}
		counter, err := m.GCounter(encoding.String("two"))
		if err != nil {
			t.Fatal(err)
		}
		if v := counter.Value(); v != 4 {
			t.Fatalf("counter.Value(): %v; want: %v", v, 4)
		}
		if d := m.Delta(); d != nil {
			t.Fatalf("m.Delta(): %v; want: %v", d, nil)
		}
		m.ResetDelta()
		if l := len(encDecState(m.State()).GetOrmap().GetEntries()); l != 2 {
			t.Fatalf("state entries len: %v; want: %v", l, 2)
		}
	})
	t.Run("should reflect a delta remove", func(t *testing.T) {
		m := NewORMap()
		m.SetGCounter(encoding.String("one"), NewGCounter())
		m.SetGCounter(encoding.String("two"), NewGCounter())
		m.ResetDelta()
		err := m.applyDelta(encDecDelta(&protocol.CrdtDelta{
			Delta: &protocol.CrdtDelta_Ormap{
				Ormap: &protocol.ORMapDelta{
					Removed: append(make([]*any.Any, 0), encoding.String("two")),
				}},
		}))
		if err != nil {
			t.Fatal(err)
		}
		if s := m.Size(); s != 1 {
			t.Fatalf("m.Size(): %v; want: %v", s, 1)
		}
		if !contains(m.Keys(), "one") {
			t.Fatalf("m.Keys() should contain 'one' but did not: %v", m.Keys())
		}
		if d := m.Delta(); d != nil {
			t.Fatalf("m.Delta(): %v; want: %v", d, nil)
		}
		m.ResetDelta()
		if l := len(encDecState(m.State()).GetOrmap().GetEntries()); l != 1 {
			t.Fatalf("state entries len: %v; want: %v", l, 1)
		}
	})
	t.Run("should reflect a delta clear", func(t *testing.T) {
		m := NewORMap()
		m.SetGCounter(encoding.String("one"), NewGCounter())
		m.SetGCounter(encoding.String("two"), NewGCounter())
		m.ResetDelta()
		err := m.applyDelta(encDecDelta(&protocol.CrdtDelta{
			Delta: &protocol.CrdtDelta_Ormap{
				Ormap: &protocol.ORMapDelta{
					Cleared: true,
				}},
		}))
		if err != nil {
			t.Fatal(err)
		}
		if s := m.Size(); s != 0 {
			t.Fatalf("m.Size(): %v; want: %v", s, 0)
		}
		if d := m.Delta(); d != nil {
			t.Fatalf("m.Delta(): %v; want: %v", d, nil)
		}
		m.ResetDelta()
		if l := len(encDecState(m.State()).GetOrmap().GetEntries()); l != 0 {
			t.Fatalf("state entries len: %v; want: %v", l, 0)
		}
	})
	t.Run("should work with protobuf keys", func(t *testing.T) {
		m := NewORMap()
		type c struct {
			Field1 string
		}
		m.SetGCounter(encoding.Struct(&c{Field1: "one"}), NewGCounter())
		m.SetGCounter(encoding.Struct(&c{Field1: "two"}), NewGCounter())
		m.ResetDelta()
		m.Delete(encoding.Struct(&c{Field1: "one"}))
		if s := m.Size(); s != 1 {
			t.Fatalf("m.Size(): %v; want: %v", s, 1)
		}
		delta := encDecDelta(m.Delta())
		if l := len(delta.GetOrmap().GetRemoved()); l != 1 {
			t.Fatalf("added len: %v; want: %v", l, 1)
		}
		c0 := &c{}
		err := encoding.DecodeStruct(m.Delta().GetOrmap().GetRemoved()[0], c0)
		if err != nil {
			t.Fatal(err)
		}
		if f1 := c0.Field1; f1 != "one" {
			t.Fatalf("c0.Field1: %v; want: %v", f1, "one")
		}
	})
	t.Run("should work with json types", func(t *testing.T) {
		m := NewORMap()
		m.SetGCounter(encoding.Struct(struct{ Foo string }{Foo: "bar"}), NewGCounter())
		m.SetGCounter(encoding.Struct(struct{ Foo string }{Foo: "baz"}), NewGCounter())
		m.ResetDelta()
		m.Delete(encoding.Struct(struct{ Foo string }{Foo: "bar"}))
		if s := m.Size(); s != 1 {
			t.Fatalf("m.Size(): %v; want: %v", s, 1)
		}
		delta := encDecDelta(m.Delta())
		if l := len(delta.GetOrmap().GetRemoved()); l != 1 {
			t.Fatalf("added len: %v; want: %v", l, 1)
		}
		c0 := &struct{ Foo string }{}
		err := encoding.DecodeStruct(m.Delta().GetOrmap().GetRemoved()[0], c0)
		if err != nil {
			t.Fatal(err)
		}
		if f1 := c0.Foo; f1 != "bar" {
			t.Fatalf("c0.Field1: %v; want: %v", f1, "bar")
		}
	})
	//t.Fatalf(": %v; want: %v", , )
}
