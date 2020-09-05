package synth

import (
	"github.com/cloudstateio/go-support/tck/proto/crdt"
	"github.com/golang/protobuf/proto"
)

func voteRequest(a ...proto.Message) *crdt.VoteRequest {
	r := &crdt.VoteRequest{
		Actions: make([]*crdt.VoteRequestAction, 0),
	}
	for _, i := range a {
		switch t := i.(type) {
		case *crdt.Get:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.VoteRequestAction{Action: &crdt.VoteRequestAction_Get{Get: t}})
		case *crdt.Delete:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.VoteRequestAction{Action: &crdt.VoteRequestAction_Delete{Delete: t}})
		case *crdt.VoteVote:
			r.Id = t.Key
			r.Actions = append(r.Actions, &crdt.VoteRequestAction{Action: &crdt.VoteRequestAction_Vote{Vote: t}})
		default:
			panic("no type matched")
		}
	}
	return r
}
