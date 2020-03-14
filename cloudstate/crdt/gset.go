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
	"hash/maphash"

	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/golang/protobuf/ptypes/any"
)

// A grow only set can have elements added to it, but not removed.
type GSet struct {
	value map[uint64]*any.Any
	added map[uint64]*any.Any
	anyHasher
}

func NewGSet() *GSet {
	return &GSet{
		value:     make(map[uint64]*any.Any),
		added:     make(map[uint64]*any.Any),
		anyHasher: anyHasher(maphash.MakeSeed()),
	}
}

func (s GSet) Size() int {
	return len(s.value)
}

func (s *GSet) Add(a *any.Any) {
	h := s.hashAny(a)
	if _, exists := s.value[h]; exists {
		return
	}
	s.value[h] = a
	s.added[h] = a
}

func (s GSet) State() *protocol.GSetState {
	return &protocol.GSetState{
		Items: s.Value(),
	}
}

func (s GSet) HasDelta() bool {
	return len(s.added) > 0
}

func (s GSet) Value() []*any.Any {
	val := make([]*any.Any, 0, s.Size())
	for _, v := range s.value {
		val = append(val, v)
	}
	return val
}

func (s GSet) Added() []*any.Any {
	val := make([]*any.Any, 0, s.Size())
	for _, v := range s.added {
		val = append(val, v)
	}
	return val
}

func (s GSet) Delta() *protocol.GSetDelta {
	return &protocol.GSetDelta{
		Added: s.Added(),
	}
}

func (s *GSet) ResetDelta() {
	s.added = make(map[uint64]*any.Any)
}

func (s *GSet) ApplyState(state *protocol.GSetState) {
	s.value = make(map[uint64]*any.Any)
	for _, v := range state.Items {
		s.value[s.hashAny(v)] = v
	}
}

func (s *GSet) ApplyDelta(delta *protocol.GSetDelta) {
	for _, v := range delta.Added {
		s.value[s.hashAny(v)] = v
	}
}
