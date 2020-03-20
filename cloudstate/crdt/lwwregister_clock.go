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

import "github.com/cloudstateio/go-support/cloudstate/protocol"

type Clock uint64

const (
	// The default clock, uses the current system time as the clock value.
	Default Clock = iota + 1

	// A reverse clock, based on the system clock.
	// Using this effectively achieves First-Write-Wins semantics.
	// This is susceptible to the same clock skew problems as the default clock.
	Reverse

	// A custom clock.
	// The custom clock value is passed by using the customClockValue parameter on
	// the {@link LWWRegister#set(Object, Clock, long)} method. The value should be a domain
	// specific monotonically increasing value. For example, if the source of the value for this
	// register is a single device, that device may attach a sequence number to each update, that
	// sequence number can be used to guarantee that the register will converge to the last update
	// emitted by that device.
	Custom

	// A custom clock, that automatically increments the custom value if the local clock value is
	// greater than it.
	//
	// <p>This is like {@link Clock#CUSTOM}, however if when performing the update in the proxy,
	// it's found that the clock value of the register is greater than the specified clock value for
	// the update, the proxy will instead use the current clock value of the register plus one.
	//
	// <p>This can guarantee that updates done on the same node will be causally ordered (addressing
	// problems caused by the system clock being adjusted), but will not guarantee causal ordering
	// for updates on different nodes, since it's possible that an update on a different node has
	// not yet been replicated to this node.

	CustomAutoIncrement
)

func fromCrdtClock(clock protocol.CrdtClock) Clock {
	switch clock {
	case protocol.CrdtClock_DEFAULT:
		return Default
	case protocol.CrdtClock_REVERSE:
		return Reverse
	case protocol.CrdtClock_CUSTOM:
		return Custom
	case protocol.CrdtClock_CUSTOM_AUTO_INCREMENT:
		return CustomAutoIncrement
	default:
		return Default
	}
}

func (c Clock) toCrdtClock() protocol.CrdtClock {
	switch c {
	case Default:
		return protocol.CrdtClock_DEFAULT
	case Reverse:
		return protocol.CrdtClock_REVERSE
	case Custom:
		return protocol.CrdtClock_CUSTOM
	case CustomAutoIncrement:
		return protocol.CrdtClock_CUSTOM_AUTO_INCREMENT
	default:
		return protocol.CrdtClock_DEFAULT
	}
}
