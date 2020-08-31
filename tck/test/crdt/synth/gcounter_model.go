package synth

import (
	"github.com/cloudstateio/go-support/tck/proto/crdt"
	"github.com/golang/protobuf/proto"
)

func gcounterRequest(a ...proto.Message) *crdt.GCounterRequest {
	r := &crdt.GCounterRequest{
		Actions: make([]*crdt.GCounterRequestAction, 0),
	}
	for _, i := range a {
		switch t := i.(type) {
		case *crdt.GCounterIncrement:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.GCounterRequestAction{Action: &crdt.GCounterRequestAction_Increment{Increment: t}})
		case *crdt.Get:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.GCounterRequestAction{Action: &crdt.GCounterRequestAction_Get{Get: t}})
		case *crdt.Delete:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.GCounterRequestAction{Action: &crdt.GCounterRequestAction_Delete{Delete: t}})
		default:
			panic("no type matched")
		}
	}
	return r
}
