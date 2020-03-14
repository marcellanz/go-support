//
// Copyright 2020 Lightbend Inc.
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

package crdt

import (
	"errors"
	"fmt"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
)

// A Vote CRDT allows all nodes an a cluster to vote on a condition, such as whether a user is online.
type Vote struct {
	selfVote        bool
	selfVoteChanged bool // delta seen
	voters          uint32
	votesFor        uint32
}

func NewVote() *Vote {
	return &Vote{
		selfVote:        false,
		selfVoteChanged: false,
		voters:          1,
		votesFor:        0,
	}
}

func (v *Vote) SelfVote() bool {
	return v.selfVote
}

func (v *Vote) Voters() uint32 {
	return v.voters
}

func (v *Vote) VotesFor() uint32 {
	return v.votesFor
}

func (v *Vote) AtLeastOne() bool {
	return v.votesFor > 0
}

func (v *Vote) Majority() bool {
	return v.votesFor > v.voters/2
}

func (v *Vote) All() bool {
	return v.votesFor == v.voters
}

func (v *Vote) Vote(vote bool) {
	if v.selfVote == vote {
		return
	}
	v.selfVoteChanged = !v.selfVoteChanged
	v.selfVote = vote
	if v.selfVote {
		v.votesFor += 1
	} else {
		v.votesFor -= 1
	}
}

func (v *Vote) HasDelta() bool {
	return v.selfVoteChanged
}

func (v *Vote) Delta() *protocol.CrdtDelta {
	if !v.selfVoteChanged {
		return nil
	}
	return &protocol.CrdtDelta{
		Delta: &protocol.CrdtDelta_Vote{Vote: &protocol.VoteDelta{
			SelfVote:    v.selfVote,
			VotesFor:    int32(v.votesFor), // TODO, we never overflow, yes?
			TotalVoters: int32(v.voters),
		}},
	}
}

func (v *Vote) ResetDelta() {
	v.selfVoteChanged = false
}

func (v *Vote) State() *protocol.CrdtState {
	return &protocol.CrdtState{
		State: &protocol.CrdtState_Vote{Vote: &protocol.VoteState{
			VotesFor:    v.votesFor,
			TotalVoters: v.voters,
			SelfVote:    v.selfVote,
		}},
	}
}

func (v *Vote) applyDelta(delta *protocol.CrdtDelta) error {
	d := delta.GetVote()
	if d == nil {
		return errors.New(fmt.Sprintf("unable to apply delta %+v to the Vote", delta))
	}
	v.voters = uint32(d.TotalVoters)
	v.votesFor = uint32(d.VotesFor)
	return nil
}

func (v *Vote) applyState(state *protocol.CrdtState) error {
	s := state.GetVote()
	if s == nil {
		return errors.New(fmt.Sprintf("unable to apply state %+v to the Vote", state))
	}
	v.selfVote = s.SelfVote
	v.votesFor = s.VotesFor
	v.voters = s.TotalVoters
	return nil
}
