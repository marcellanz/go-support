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
	"fmt"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
)

type Flag struct {
	value bool
	delta bool
}

var _ CRDT = (*Flag)(nil)

func NewFlag() *Flag {
	return &Flag{}
}

func (f Flag) Value() bool {
	return f.value
}

func (f *Flag) Enable() {
	if !f.value {
		f.value, f.delta = true, true
	}
}

func (f Flag) Delta() *protocol.CrdtDelta {
	return &protocol.CrdtDelta{
		Delta: &protocol.CrdtDelta_Flag{
			Flag: &protocol.FlagDelta{
				Value: f.delta,
			},
		},
	}
}

func (f *Flag) HasDelta() bool {
	return f.delta
}

func (f *Flag) resetDelta() {
	f.delta = false
}

func (f *Flag) applyDelta(delta *protocol.CrdtDelta) error {
	d := delta.GetFlag()
	if d == nil {
		return fmt.Errorf("unable to apply delta %+v to Flag", delta)
	}
	f.value = f.value || d.Value
	return nil
}

func (f Flag) State() *protocol.CrdtState {
	return &protocol.CrdtState{
		State: &protocol.CrdtState_Flag{
			Flag: &protocol.FlagState{
				Value: f.value,
			},
		},
	}
}

func (f *Flag) applyState(state *protocol.CrdtState) error {
	s := state.GetFlag()
	if s == nil {
		return fmt.Errorf("unable to apply state %+v to Flag", state)
	}
	f.value = s.GetValue()
	return nil
}
