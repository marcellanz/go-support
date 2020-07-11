package crdt

import (
	"reflect"
	"testing"

	"github.com/cloudstateio/go-support/cloudstate/protocol"
)

func Test_newFor(t *testing.T) {
	type args struct {
		state *protocol.CrdtState
	}
	tests := []struct {
		name string
		args args
		want CRDT
	}{
		{"newForFlag", args{state: &protocol.CrdtState{State: &protocol.CrdtState_Flag{}}}, NewFlag()},
		{"newForGCounter", args{state: &protocol.CrdtState{State: &protocol.CrdtState_Gcounter{}}}, NewGCounter()},
		{"newForGset", args{state: &protocol.CrdtState{State: &protocol.CrdtState_Gset{}}}, NewGSet()},
		{"newForLWWRegister", args{state: &protocol.CrdtState{State: &protocol.CrdtState_Lwwregister{}}}, NewLWWRegister(nil)},
		{"newForORMap", args{state: &protocol.CrdtState{State: &protocol.CrdtState_Ormap{}}}, NewORMap()},
		{"newForORSet", args{state: &protocol.CrdtState{State: &protocol.CrdtState_Orset{}}}, NewORSet()},
		{"newForPNCounter", args{state: &protocol.CrdtState{State: &protocol.CrdtState_Pncounter{}}}, NewPNCounter()},
		{"newForVote", args{state: &protocol.CrdtState{State: &protocol.CrdtState_Vote{}}}, NewVote()},
		{"newForNil", args{state: &protocol.CrdtState{State: nil}}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newFor(tt.args.state); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newFor() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
