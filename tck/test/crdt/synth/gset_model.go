package synth

import (
	"github.com/cloudstateio/go-support/tck/proto/crdt"
	"github.com/golang/protobuf/proto"
)

func gsetRequest(a ...proto.Message) *crdt.GSetRequest {
	r := &crdt.GSetRequest{
		Actions: make([]*crdt.GSetRequestAction, 0),
	}
	for _, i := range a {
		switch t := i.(type) {
		case *crdt.Get:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.GSetRequestAction{Action: &crdt.GSetRequestAction_Get{Get: t}})
		case *crdt.Delete:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.GSetRequestAction{Action: &crdt.GSetRequestAction_Delete{Delete: t}})
		case *crdt.GSetAdd:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.GSetRequestAction{Action: &crdt.GSetRequestAction_Add{Add: t}})
		default:
			panic("no type matched")
		}
	}
	return r
}
