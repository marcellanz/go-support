package synth

import (
	"github.com/cloudstateio/go-support/tck/proto/crdt"
	"github.com/golang/protobuf/proto"
)

func lwwRegisterRequest(a ...proto.Message) *crdt.LWWRegisterRequest {
	r := &crdt.LWWRegisterRequest{
		Actions: make([]*crdt.LWWRegisterRequestAction, 0),
	}
	for _, i := range a {
		switch t := i.(type) {
		case *crdt.Get:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.LWWRegisterRequestAction{Action: &crdt.LWWRegisterRequestAction_Get{Get: t}})
		case *crdt.Delete:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.LWWRegisterRequestAction{Action: &crdt.LWWRegisterRequestAction_Delete{Delete: t}})
		case *crdt.LWWRegisterSet:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.LWWRegisterRequestAction{Action: &crdt.LWWRegisterRequestAction_Set{Set: t}})
		case *crdt.LWWRegisterSetWithClock:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.LWWRegisterRequestAction{Action: &crdt.LWWRegisterRequestAction_SetWithClock{SetWithClock: t}})
		default:
			panic("no type matched")
		}
	}
	return r
}
