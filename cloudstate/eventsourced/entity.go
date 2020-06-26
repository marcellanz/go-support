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

import (
	"github.com/golang/protobuf/proto"
)

type (
	ServiceName string
	EntityId    string
	CommandId   int64
)

func (sn ServiceName) String() string {
	return string(sn)
}

func (id CommandId) Value() int64 {
	return int64(id)
}

// Entity describes an event sourced entity. It is used to be registered as
// an event sourced entity on a CloudState instance.
type Entity struct {
	// ServiceName is the fully qualified name of the service that implements this entities interface.
	// Setting it is mandatory.
	ServiceName ServiceName
	// PersistenceID is used to namespace events in the journal, useful for
	// when you share the same database between multiple entities. It defaults to
	// the simple name for the entity type.
	// It’s good practice to select one explicitly, this means your database
	// isn’t depend on type names in your code.
	// Setting it is mandatory.
	PersistenceID string
	// The snapshotEvery parameter controls how often snapshots are taken,
	// so that the entity doesn't need to be recovered from the whole journal
	// each time it’s loaded. If left unset, it defaults to 100.
	// Setting it to a negative number will result in snapshots never being taken.
	SnapshotEvery int64
	// EntityFactory is a factory method which generates a new Entity.

	EntityFunc          func(id EntityId) interface{}
	CommandFunc         func(entity interface{}, ctx *Context, name string, msg proto.Message) (reply proto.Message, err error)
	EventFunc           func(entity interface{}, ctx *Context, event interface{}) error
	SnapshotFunc        func(entity interface{}, ctx *Context) (snapshot interface{}, err error)
	SnapshotHandlerFunc func(entity interface{}, ctx *Context, snapshot interface{}) error
}
