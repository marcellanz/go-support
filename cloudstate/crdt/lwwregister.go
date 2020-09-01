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
	"github.com/golang/protobuf/ptypes/any"
)

// LWWRegister, or Last-Write-Wins Register, is a CRDT that can hold any value,
// along with a clock value and node id to indicate when it was updated by which
// node. If two nodes have two different versions of the value, the one with the
// highest clock value wins. If the clock values are equal, then a stable function
// on the nodes is used to determine it (eg, the node with the lowest address).
// Note that LWWRegisters do not support partial updates of their values. If the
// register holds a person object, and one node updates the age property, while
// another concurrently updates the name property, only one of those updates will
// eventually win. By default, LWWRegister’s are vulnerable to clock skew between
// nodes. Cloudstate supports optionally providing a custom clock value should a
// more trustworthy ordering for updates be available.
type LWWRegister struct {
	value            *any.Any
	clock            Clock
	customClockValue int64
	delta            lwwRegisterDelta
}

type lwwRegisterDelta struct {
	value            *any.Any
	clock            Clock
	customClockValue int64
}

var _ CRDT = (*LWWRegister)(nil)

func NewLWWRegister(x *any.Any) *LWWRegister {
	return NewLWWRegisterWithClock(x, Default, 0)
}

func NewLWWRegisterWithClock(x *any.Any, c Clock, customClockValue int64) *LWWRegister {
	return &LWWRegister{
		value:            x,
		clock:            c,
		customClockValue: customClockValue,
		delta:            lwwRegisterDelta{},
	}
}

func (r *LWWRegister) Value() *any.Any {
	return r.value
}

func (r *LWWRegister) Set(x *any.Any) {
	r.SetWithClock(x, Default, 0)
}

// The custom clock value to use if the clock selected is a custom clock.
// This is ignored if the clock is not a custom clock
func (r *LWWRegister) SetWithClock(x *any.Any, c Clock, customClockValue int64) {
	r.value = x
	r.clock = c
	r.customClockValue = customClockValue
	r.delta = lwwRegisterDelta{
		value:            x,
		clock:            c,
		customClockValue: customClockValue,
	}
}

func (r *LWWRegister) Delta() *protocol.CrdtDelta {
	return &protocol.CrdtDelta{
		Delta: &protocol.CrdtDelta_Lwwregister{
			Lwwregister: &protocol.LWWRegisterDelta{
				Value:            r.delta.value,
				Clock:            r.delta.clock.toCrdtClock(),
				CustomClockValue: r.delta.customClockValue,
			},
		},
	}

}

func (r *LWWRegister) HasDelta() bool {
	return r.delta.value != nil
}

func (r *LWWRegister) resetDelta() {
	r.delta = lwwRegisterDelta{}
	r.clock = Default
	r.customClockValue = 0
}

func (r *LWWRegister) applyDelta(delta *protocol.CrdtDelta) error {
	d := delta.GetLwwregister()
	if d == nil {
		return fmt.Errorf("unable to apply state %+v to LWWRegister", delta)
	}
	r.value = d.GetValue()
	return nil
}

func (r *LWWRegister) State() *protocol.CrdtState {
	return &protocol.CrdtState{
		State: &protocol.CrdtState_Lwwregister{
			Lwwregister: &protocol.LWWRegisterState{
				Value:            r.value,
				Clock:            r.clock.toCrdtClock(),
				CustomClockValue: r.customClockValue,
			},
		},
	}
}

func (r *LWWRegister) applyState(delta *protocol.CrdtState) error {
	d := delta.GetLwwregister()
	if d == nil {
		return fmt.Errorf("unable to apply delta %+v to LWWRegister", delta)
	}
	r.value = d.GetValue()
	r.clock = fromCrdtClock(d.GetClock())
	r.customClockValue = d.GetCustomClockValue()
	return nil
}
