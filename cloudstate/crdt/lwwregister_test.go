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
)

func TestLWWRegister(t *testing.T) {
	type Example struct {
		Field1 string
	}

	t.Run("should be instantiated with a value", func(t *testing.T) {
		r := NewLWWRegister(encoding.Struct(Example{Field1: "foo"}))
		example := Example{}
		err := encoding.UnmarshalJSON(r.Value(), &example)
		if err != nil {
			t.Error(err)
		}
		if example.Field1 != "foo" {
			t.Errorf("example.Field1: %v; want: %v", example.Field1, "foo")
		}
		if r.HasDelta() {
			t.Errorf("register has lwwRegisterDelta but should not")
		}
		state := lwwREncDecState(r.State())
		err = encoding.UnmarshalJSON(state.Value, &example)
		if err != nil {
			t.Error(err)
		}
		r.ResetDelta()
		if example.Field1 != "foo" {
			t.Errorf("example.Field1: %v; want: %v", example.Field1, "foo")
		}
		if r.clock != Default {
			t.Errorf("r.clock: %v; want: %v", r.clock, Default)
		}
	})

	t.Run("should reflect a state update", func(t *testing.T) {
		r := LWWRegister{
			value: encoding.Struct(Example{Field1: "bar"}),
		}
		r.ApplyState(lwwREncDecState(&protocol.LWWRegisterState{
			Value: encoding.Struct(Example{Field1: "foo"}),
		}))
		example := Example{}
		err := encoding.UnmarshalJSON(r.Value(), &example)
		if err != nil {
			t.Error(err)
		}
		if example.Field1 != "foo" {
			t.Errorf("example.Field1: %v; want: %v", example.Field1, "foo")
		}
	})

	t.Run("should generate a lwwRegisterDelta", func(t *testing.T) {
		r := NewLWWRegister(encoding.Struct(Example{Field1: "foo"}))
		r.Set(encoding.Struct(Example{Field1: "bar"}))
		example := Example{}
		err := encoding.UnmarshalJSON(r.value, &example)
		if err != nil {
			t.Error(err)
		}
		if example.Field1 != "bar" {
			t.Errorf("example.Field1: %v; want: %v", example.Field1, "bar")
		}
		d := lwwREncDecDelta(r.Delta())
		r.ResetDelta()
		e := Example{}
		err = encoding.UnmarshalJSON(d.GetValue(), &e)
		if err != nil {
			t.Error(err)
		}
		if example.Field1 != "bar" {
			t.Errorf("example.Field1: %v; want: %v", example.Field1, "bar")
		}
		if r.clock != Default {
			t.Errorf("r.clock: %v; want: %v", r.clock, Default)
		}
		if r.HasDelta() {
			t.Errorf("register has lwwRegisterDelta but should not")
		}
	})

	t.Run("should generate deltas with a custom clock", func(t *testing.T) {
		r := NewLWWRegister(encoding.Struct(Example{Field1: "foo"}))
		r.SetWithClock(encoding.Struct(Example{Field1: "bar"}), Custom, 10)
		example := Example{}
		err := encoding.UnmarshalJSON(r.value, &example)
		if err != nil {
			t.Error(err)
		}
		if example.Field1 != "bar" {
			t.Errorf("example.Field1: %v; want: %v", example.Field1, "bar")
		}
		d := lwwREncDecDelta(r.Delta())
		r.ResetDelta()
		e := Example{}
		err = encoding.UnmarshalJSON(d.GetValue(), &e)
		if err != nil {
			t.Error(err)
		}
		if example.Field1 != "bar" {
			t.Fatalf("example.Field1: %v; want: %v", example.Field1, "bar")
		}
		if d.Clock != Custom.toCrdtClock() {
			t.Fatalf("r.clock: %v; want: %v", d.Clock, Custom)
		}
		if d.CustomClockValue != 10 {
			t.Fatalf("r.customClockValue: %v; want: %v", d.CustomClockValue, 10)
		}
		if r.HasDelta() {
			t.Fatalf("register has lwwRegisterDelta but should not")
		}
	})

	t.Run("should reflect a lwwRegisterDelta update", func(t *testing.T) {
		r := NewLWWRegister(encoding.Struct(Example{Field1: "foo"}))
		//r := LWWRegister{}
		//r.Set(encoding.Struct(Example{Field1: "foo"})) // TODO: this is not the same, check

		r.ApplyDelta(lwwREncDecDelta(&protocol.LWWRegisterDelta{
			Value: encoding.Struct(Example{Field1: "bar"}),
		}))
		e := Example{}
		err := encoding.UnmarshalJSON(r.Value(), &e)
		if err != nil {
			t.Fatal(err)
		}
		if e.Field1 != "bar" {
			t.Fatalf("example.Field1: %v; want: %v", e.Field1, "bar")
		}
		if r.HasDelta() {
			t.Fatalf("register has lwwRegisterDelta but should not")
		}
		e2 := Example{}
		err = encoding.UnmarshalJSON(lwwREncDecState(r.State()).GetValue(), &e2)
		if err != nil {
			t.Fatal(err)
		}
		if e2.Field1 != "bar" {
			t.Fatalf("example.Field1: %v; want: %v", e.Field1, "bar")
		}
	})

	t.Run("should work with primitive types", func(t *testing.T) {
		r := NewLWWRegister(encoding.String("momo"))
		state := lwwREncDecState(r.State())
		r.ResetDelta()
		stateValue := encoding.DecodeString(state.GetValue())
		if stateValue != "momo" {
			t.Fatalf("stateValue: %v; want: %v", stateValue, "momo")
		}
		r.Set(encoding.String("hello"))
		rValue := encoding.DecodeString(r.Value())
		if rValue != "hello" {
			t.Fatalf("r.Value(): %v; want: %v", rValue, "hello")
		}
	})
}

func lwwREncDecState(s *protocol.LWWRegisterState) *protocol.LWWRegisterState {
	marshal, err := proto.Marshal(s)
	if err != nil {
		// we panic for convenience
		panic(err)
	}
	out := &protocol.LWWRegisterState{}
	if err := proto.Unmarshal(marshal, out); err != nil {
		panic(err)
	}
	return out
}

func lwwREncDecDelta(s *protocol.LWWRegisterDelta) *protocol.LWWRegisterDelta {
	marshal, err := proto.Marshal(s)
	if err != nil {
		// we panic for convenience
		panic(err)
	}
	out := &protocol.LWWRegisterDelta{}
	if err := proto.Unmarshal(marshal, out); err != nil {
		panic(err)
	}
	return out
}
