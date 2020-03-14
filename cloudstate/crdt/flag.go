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
	"github.com/cloudstateio/go-support/cloudstate/protocol"
)

type Flag struct {
	value bool
	delta bool
}

func (f Flag) Value() bool {
	return f.value
}

func (f *Flag) Enable() {
	if !f.value {
		f.value = true
		f.delta = true
	}
}

func (f Flag) Delta() *protocol.FlagDelta {
	return &protocol.FlagDelta{
		Value: true,
	}
}

func (f Flag) HasDelta() bool {
	return f.delta
}

func (f *Flag) ResetDelta() {
	f.delta = false
}

func (f *Flag) ApplyDelta(d *protocol.FlagDelta) {
	f.value = f.value || d.Value
}

func (f Flag) State() *protocol.FlagState {
	return &protocol.FlagState{
		Value: f.value,
	}
}

func (f *Flag) ApplyState(s *protocol.FlagState) {
	f.value = s.Value
}
