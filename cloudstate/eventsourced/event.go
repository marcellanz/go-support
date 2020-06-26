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

package eventsourced

type OnNext func(event interface{}) error
type OnErr func(err error)
type Subscription struct {
	OnNext
	OnErr
	active bool
}

func (s *Subscription) Unsubscribe() {
	s.active = false
}

type EventEmitter interface {
	Emit(event interface{})
	Subscribe(subs *Subscription) *Subscription
	Events() []interface{}
	Clear()
}

func newEmitter() *eventEmitter {
	return &eventEmitter{
		events:        make([]interface{}, 0),
		subscriptions: make([]*Subscription, 0),
	}
}

type eventEmitter struct {
	events        []interface{}
	subscriptions []*Subscription
}

// Emit will immediately invoke the associated event handler for that event.
// This both validates that the event can be applied to the current state, as well as
// updates the state so that subsequent processing in the command handler can use it.
func (e *eventEmitter) Emit(event interface{}) {
	for _, s := range e.subscriptions {
		if !s.active {
			continue
		}
		if err := s.OnNext(event); err != nil {
			s.OnErr(err)
		}
	}
	e.events = append(e.events, event)
}

func (e *eventEmitter) Events() []interface{} {
	return e.events
}

func (e *eventEmitter) Subscribe(s *Subscription) *Subscription {
	s.active = true
	e.subscriptions = append(e.subscriptions, s)
	return s
}

func (e *eventEmitter) Clear() {
	e.events = make([]interface{}, 0)
}
