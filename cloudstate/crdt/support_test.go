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
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
)

func contains(x []*any.Any, s ...string) bool {
	n := 0
	for _, a0 := range x {
		sa0 := encoding.DecodeString(a0)
		for _, s0 := range s {
			if sa0 == s0 {
				n++
			}
			if n == len(s) {
				return true
			}
		}
	}
	return false
}

func encDecState(s *protocol.CrdtState) *protocol.CrdtState {
	marshal, err := proto.Marshal(s)
	if err != nil {
		// we panic for convenience in test
		panic(err)
	}
	out := &protocol.CrdtState{}
	if err := proto.Unmarshal(marshal, out); err != nil {
		panic(err)
	}
	return out
}

func encDecDelta(s *protocol.CrdtDelta) *protocol.CrdtDelta {
	marshal, err := proto.Marshal(s)
	if err != nil {
		// we panic for convenience in test
		panic(err)
	}
	out := &protocol.CrdtDelta{}
	if err := proto.Unmarshal(marshal, out); err != nil {
		panic(err)
	}
	return out
}
