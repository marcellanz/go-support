package synth

import (
	"github.com/cloudstateio/go-support/tck/proto/crdt"
	"github.com/golang/protobuf/proto"
)

func flagRequest(a ...proto.Message) *crdt.FlagRequest {
	r := &crdt.FlagRequest{
		Actions: make([]*crdt.FlagRequestAction, 0),
	}
	for _, i := range a {
		switch t := i.(type) {
		case *crdt.Get:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.FlagRequestAction{Action: &crdt.FlagRequestAction_Get{Get: t}})
		case *crdt.Delete:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.FlagRequestAction{Action: &crdt.FlagRequestAction_Delete{Delete: t}})
		case *crdt.FlagEnable:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.FlagRequestAction{Action: &crdt.FlagRequestAction_Enable{Enable: t}})
		default:
			panic("no type matched")
		}
	}
	return r
}
