package synth

import (
	"github.com/cloudstateio/go-support/tck/proto/crdt"
	"github.com/golang/protobuf/proto"
)

func orsetRequest(a ...proto.Message) *crdt.ORSetRequest {
	r := &crdt.ORSetRequest{
		Actions: make([]*crdt.ORSetRequestAction, 0),
	}
	for _, i := range a {
		switch t := i.(type) {
		case *crdt.Get:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.ORSetRequestAction{Action: &crdt.ORSetRequestAction_Get{Get: t}})
		case *crdt.Delete:
			r.Actions = append(r.Actions, &crdt.ORSetRequestAction{Action: &crdt.ORSetRequestAction_Delete{Delete: t}})
			r.Id = t.Key
		case *crdt.ORSetAdd:
			r.Actions = append(r.Actions, &crdt.ORSetRequestAction{Action: &crdt.ORSetRequestAction_Add{Add: t}})
			r.Id = t.Key
		case *crdt.ORSetRemove:
			r.Actions = append(r.Actions, &crdt.ORSetRequestAction{Action: &crdt.ORSetRequestAction_Remove{Remove: t}})
			r.Id = t.Key
		default:
			panic("no type matched")
		}
	}
	return r
}
