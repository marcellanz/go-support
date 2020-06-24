package eventsourced

import (
	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/golang/protobuf/ptypes/any"
)

// marshalEventsAny receives the events emitted through the handling of a command
// and marshals them to the event serialized form.
func MarshalEventsAny(entityContext *Context) ([]*any.Any, error) {
	events := make([]*any.Any, 0)
	for _, evt := range entityContext.Events() {
		event, err := encoding.MarshalAny(evt)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	entityContext.Clear()
	return events, nil
}
