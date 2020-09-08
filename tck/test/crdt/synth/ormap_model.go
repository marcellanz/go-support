package synth

import (
	"github.com/cloudstateio/go-support/tck/proto/crdt"
	"github.com/golang/protobuf/proto"
)

func ormapRequest(a ...proto.Message) *crdt.ORMapRequest {
	r := &crdt.ORMapRequest{
		Actions: make([]*crdt.ORMapRequestAction, 0),
	}
	for _, i := range a {
		switch t := i.(type) {
		case *crdt.Get:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.ORMapRequestAction{Action: &crdt.ORMapRequestAction_Get{Get: t}})
		case *crdt.Delete:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.ORMapRequestAction{Action: &crdt.ORMapRequestAction_Delete{Delete: t}})
		case *crdt.ORMapSet:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.ORMapRequestAction{Action: &crdt.ORMapRequestAction_SetKey{SetKey: t}})
		case *crdt.ORMapDelete:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.ORMapRequestAction{Action: &crdt.ORMapRequestAction_DeleteKey{DeleteKey: t}})
		case *crdt.ORMapActionRequest:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.ORMapRequestAction{Action: &crdt.ORMapRequestAction_Request{Request: t}})
		default:
			panic("no type matched")
		}
	}
	return r
}
