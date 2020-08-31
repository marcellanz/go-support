package synth

import (
	"github.com/cloudstateio/go-support/tck/proto/crdt"
	"github.com/golang/protobuf/proto"
)

func pncounterRequest(a ...proto.Message) *crdt.PNCounterRequest {
	r := &crdt.PNCounterRequest{
		Actions: make([]*crdt.PNCounterRequestAction, 0, len(a)),
	}
	for _, i := range a {
		switch t := i.(type) {
		case *crdt.PNCounterIncrement:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.PNCounterRequestAction{Action: &crdt.PNCounterRequestAction_Increment{Increment: t}})
		case *crdt.PNCounterDecrement:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.PNCounterRequestAction{Action: &crdt.PNCounterRequestAction_Decrement{Decrement: t}})
		case *crdt.Get:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.PNCounterRequestAction{Action: &crdt.PNCounterRequestAction_Get{Get: t}})
		case *crdt.Delete:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.PNCounterRequestAction{Action: &crdt.PNCounterRequestAction_Delete{Delete: t}})
		default:
			panic("no type matched")
		}
	}
	return r
}
