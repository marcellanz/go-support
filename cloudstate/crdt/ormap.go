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
	"errors"
	"fmt"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/golang/protobuf/ptypes/any"
)

type ORMap struct {
	value map[uint64]*orMapValue
	delta orMapDelta
	*anyHasher
}

type orMapValue struct {
	key   *any.Any
	value CRDT
}

var _ CRDT = (*ORMap)(nil)

type orMapDelta struct {
	added   map[uint64]*any.Any
	removed map[uint64]*any.Any
	cleared bool
}

func NewORMap() *ORMap {
	return &ORMap{
		value: make(map[uint64]*orMapValue),
		delta: orMapDelta{
			added:   make(map[uint64]*any.Any, 0),
			removed: make(map[uint64]*any.Any, 0),
			cleared: false,
		},
		anyHasher: &anyHasher{},
	}
}

func (m *ORMap) HasKey(x *any.Any) (hasKey bool) {
	_, hasKey = m.value[m.hashAny(x)]
	return
}

func (m *ORMap) Size() int {
	return len(m.value)
}

func (m *ORMap) Values() []*protocol.CrdtState {
	values := make([]*protocol.CrdtState, 0, len(m.value))
	for _, v := range m.value {
		values = append(values, v.value.State())
	}
	return values
}

func (m *ORMap) Keys() []*any.Any {
	keys := make([]*any.Any, 0, len(m.value))
	for _, v := range m.value {
		keys = append(keys, v.key)
	}
	return keys
}

func (m *ORMap) Get(key *any.Any) *protocol.CrdtState {
	if s, ok := m.value[m.hashAny(key)]; ok {
		return s.value.State()
	}
	return nil
}

func (m *ORMap) set(key *any.Any, value CRDT) {
	hk := m.hashAny(key)
	// why that?
	if _, has := m.value[hk]; has {
		if _, has := m.delta.added[hk]; !has {
			m.delta.removed[hk] = key
		}
	}
	m.value[hk] = &orMapValue{
		key:   key,
		value: value,
	}
	m.delta.added[hk] = key
}

func (m *ORMap) Delete(key *any.Any) {
	hkey := m.hashAny(key)
	if _, has := m.value[hkey]; has {
		if len(m.value) == 1 {
			m.Clear()
		} else {
			delete(m.value, hkey)
			if _, has := m.delta.added[hkey]; has {
				delete(m.delta.added, hkey)
			} else {
				m.delta.removed[hkey] = key
			}
		}
	}
}

func (d *orMapDelta) clear() {
	d.cleared = true
	d.added = make(map[uint64]*any.Any)
	d.removed = make(map[uint64]*any.Any)
}

func (m *ORMap) Clear() {
	if len(m.value) > 0 {
		m.value = make(map[uint64]*orMapValue)
		m.delta.clear()
	}
}

func (m *ORMap) HasDelta() bool {
	if m.delta.cleared || len(m.delta.added) > 0 || len(m.delta.removed) > 0 {
		return true
	}
	for _, v := range m.value {
		if v.value.HasDelta() {
			return true
		}
	}
	return false
}

func (m *ORMap) Delta() *protocol.CrdtDelta {
	if !m.HasDelta() {
		return nil
	}
	updated := make([]*protocol.ORMapEntryDelta, 0)
	added := make([]*protocol.ORMapEntry, 0)
	for _, v := range m.value {
		if _, has := m.delta.added[m.hashAny(v.key)]; has {
			added = append(added, &protocol.ORMapEntry{
				Key:   v.key,
				Value: v.value.State(),
			})
		} else {
			if v.value.HasDelta() {
				updated = append(updated, &protocol.ORMapEntryDelta{
					Key:   v.key,
					Delta: v.value.Delta(),
				})
			}
		}
	}
	removed := make([]*any.Any, 0, len(m.delta.removed))
	for _, e := range m.delta.removed {
		removed = append(removed, e)
	}
	return &protocol.CrdtDelta{
		Delta: &protocol.CrdtDelta_Ormap{
			Ormap: &protocol.ORMapDelta{
				Cleared: m.delta.cleared,
				Removed: removed,
				Updated: updated,
				Added:   added,
			},
		},
	}
}

func (m *ORMap) applyDelta(delta *protocol.CrdtDelta) error {
	d := delta.GetOrmap()
	if d == nil {
		return errors.New(fmt.Sprintf("unable to apply delta %v to the ORMap", delta))
	}
	if d.GetCleared() {
		m.value = make(map[uint64]*orMapValue)
	}
	for _, r := range d.GetRemoved() {
		if m.HasKey(r) {
			delete(m.value, m.hashAny(r))
		}
	}
	for _, a := range d.Added {
		if m.HasKey(a.GetKey()) {
			continue
		}
		switch a.GetValue().GetState().(type) {
		case *protocol.CrdtState_Gcounter:
			crdt := NewGCounter()
			if err := crdt.applyState(a.GetValue()); err != nil {
				return err
			}
			m.value[m.hashAny(a.GetKey())] = &orMapValue{
				key:   a.GetKey(),
				value: crdt,
			}
		}
	}
	for _, u := range d.Updated {
		if v, has := m.value[m.hashAny(u.GetKey())]; has {
			if err := v.value.applyDelta(u.GetDelta()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *ORMap) resetDelta() {
	for _, v := range m.value {
		v.value.resetDelta()
	}
	m.delta.cleared = false // whats the thing with cleared to be different to orMapDelta.clear()?
	m.delta.added = make(map[uint64]*any.Any)
	m.delta.removed = make(map[uint64]*any.Any)
}

func (m *ORMap) State() *protocol.CrdtState {
	entries := make([]*protocol.ORMapEntry, 0, len(m.value))
	for _, v := range m.value {
		entries = append(entries, &protocol.ORMapEntry{
			Key:   v.key,
			Value: v.value.State(),
		})
	}
	return &protocol.CrdtState{
		State: &protocol.CrdtState_Ormap{
			Ormap: &protocol.ORMapState{
				Entries: entries,
			},
		},
	}
}

func (m *ORMap) applyState(state *protocol.CrdtState) error {
	s := state.GetOrmap()
	if s == nil {
		return fmt.Errorf("unable to apply state %v to the ORMap", state)
	}
	m.value = make(map[uint64]*orMapValue)
	for _, entry := range s.GetEntries() {
		v := &orMapValue{
			key:   entry.GetKey(),
			value: newFor(entry.GetValue()),
		}
		if err := v.value.applyState(entry.GetValue()); err != nil {
			return err
		}
		m.value[m.hashAny(v.key)] = v
	}
	return nil
}
