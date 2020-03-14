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

func TestFlag(t *testing.T) {
	t.Run("should be disabled when instantiated", func(t *testing.T) {
		f := Flag{}
		if f.Value() {
			t.Errorf("flag should be false but was not")
		}
	})
	t.Run("should reflect a state update", func(t *testing.T) {
		f := Flag{}
		f.ApplyState(flagEncDecState(&protocol.FlagState{
			Value: true,
		}))
		if !f.Value() {
			t.Errorf("flag should be true but was not")
		}
	})
	t.Run("should reflect a lwwRegisterDelta update", func(t *testing.T) {
		f := Flag{}
		f.ApplyDelta(&protocol.FlagDelta{
			Value: true,
		})
		if !f.Value() {
			t.Errorf("flag should be true but was not")
		}
	})
	t.Run("should generate deltas", func(t *testing.T) {
		f := Flag{}
		f.Enable()
		if !flagEncDecDelta(f.Delta()).Value {
			t.Errorf("lwwRegisterDelta should be true but was not")
		}
		f.ResetDelta()
		if f.HasDelta() {
			t.Errorf("flag should have no lwwRegisterDelta")
		}
	})
	t.Run("should return its state", func(t *testing.T) {
		f := Flag{}
		if flagEncDecState(f.State()).Value {
			t.Errorf("value should be false but was not")
		}
		f.ResetDelta()
		f.Enable()
		if !flagEncDecState(f.State()).Value {
			t.Errorf("lwwRegisterDelta should be true but was not")
		}
		f.ResetDelta()
		if f.HasDelta() {
			t.Errorf("flag should have no lwwRegisterDelta")
		}
	})
}

func flagEncDecState(s *protocol.FlagState) *protocol.FlagState {
	b := make([]byte, 0)[:]
	marshal, err := s.XXX_Marshal(b, true)
	if err != nil {
		// we panic for convenience
		panic(err)
	}
	out := &protocol.FlagState{}
	if err := out.XXX_Unmarshal(marshal); err != nil {
		panic(err)
	}
	return out
}

func flagEncDecDelta(s *protocol.FlagDelta) *protocol.FlagDelta {
	b := make([]byte, 0)[:]
	marshal, err := s.XXX_Marshal(b, true)
	if err != nil {
		// we panic for convenience
		panic(err)
	}
	out := &protocol.FlagDelta{}
	if err := out.XXX_Unmarshal(marshal); err != nil {
		panic(err)
	}
	return out
}
