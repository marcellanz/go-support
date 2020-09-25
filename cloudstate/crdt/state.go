//
// Copyright 2019 Lightbend Inc.
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

func newFor(state *protocol.CrdtState) CRDT {
	switch state.GetState().(type) {
	case *protocol.CrdtState_Flag:
		return NewFlag()
	case *protocol.CrdtState_Gcounter:
		return NewGCounter()
	case *protocol.CrdtState_Gset:
		return NewGSet()
	case *protocol.CrdtState_Lwwregister:
		return NewLWWRegister(nil)
	case *protocol.CrdtState_Ormap:
		return NewORMap()
	case *protocol.CrdtState_Orset:
		return NewORSet()
	case *protocol.CrdtState_Pncounter:
		return NewPNCounter()
	case *protocol.CrdtState_Vote:
		return NewVote()
	default:
		return nil
	}
}
