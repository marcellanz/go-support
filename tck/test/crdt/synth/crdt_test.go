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

package synth

import (
	"context"
	"testing"
	"time"

	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	crdt3 "github.com/cloudstateio/go-support/tck/proto/crdt"
)

func TestCRDT(t *testing.T) {
	s := newServer(t)
	s.newClientConn()
	defer s.teardown()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("Entity Discovery should find the SyntheticCRDTs service", func(t *testing.T) {
		edc := protocol.NewEntityDiscoveryClient(s.conn)
		discover, err := edc.Discover(ctx, &protocol.ProxyInfo{
			ProtocolMajorVersion: 1,
			ProtocolMinorVersion: 0,
			ProxyName:            "a-cs-proxy",
			ProxyVersion:         "0.0.0",
			SupportedEntityTypes: []string{protocol.EventSourced, protocol.CRDT},
		})
		if err != nil {
			t.Fatal(err)
		}
		if got, want := len(discover.GetEntities()), 1; got != want {
			t.Fatalf("discover.Entities is:%d, should be: %d", got, want)
		}
		if got, want := discover.GetEntities()[0].GetServiceName(), serviceName; got != want {
			t.Fatalf("serviceName = %s; want: %s", got, want)
		}
	})

	t.Run("GCounter", func(t *testing.T) {
		p := newProxy(ctx, s)
		t.Run("send a CrdtInit should not fail", func(t *testing.T) {
			p.sendInit(&protocol.CrdtInit{
				ServiceName: serviceName,
				EntityId:    "gcounter-1",
			})
		})
		t.Run("incrementing a GCounter should emit a client action and create state action", func(t *testing.T) {
			out := p.sendRecvCmd(command{
				&protocol.Command{EntityId: "gcounter-1", Name: "IncrementGCounter"},
				&crdt3.GCounterIncrement{Key: "gcounter-1", Value: 7},
			})
			switch m := out.Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				// we expect a client action
				value := crdt3.GCounterValue{}
				err := encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				if err != nil {
					t.Fatal(err)
				}
				if got, want := value.GetValue(), uint64(7); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
				}
				// and we expect a create state action
				if got, want := m.Reply.GetStateAction().GetCreate().GetGcounter().GetValue(), uint64(7); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, out.GetReply())
				}
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
		t.Run("a second increment should emit a client action and an update state action", func(t *testing.T) {
			out := p.sendRecvCmd(command{
				&protocol.Command{EntityId: "gcounter-1", Name: "IncrementGCounter"},
				&crdt3.GCounterIncrement{Key: "gcounter-1", Value: 7},
			})
			switch m := out.Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				err := encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				if err != nil {
					t.Fatal(err)
				}
				if got, want := value.GetValue(), uint64(14); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
				}
				if got, want := m.Reply.GetStateAction().GetUpdate().GetGcounter().GetIncrement(), uint64(7); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, m.Reply)
				}
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
		t.Run("calling GetGCounter should return the counters value", func(t *testing.T) {
			out := p.sendRecvCmd(
				command{
					&protocol.Command{EntityId: "gcounter-1", Name: "GetGCounter"},
					&crdt3.Get{Key: "gcounter-1"},
				},
			)
			switch m := out.Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				err := encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				if err != nil {
					t.Fatal(err)
				}
				if got, want := value.GetValue(), uint64(14); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
				}
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
		t.Run("the counter should apply state and return its value", func(t *testing.T) {
			err := p.sendState(state{
				s: &protocol.CrdtState{
					State: &protocol.CrdtState_Gcounter{Gcounter: &protocol.GCounterState{
						Value: uint64(21)},
					},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			out := p.sendRecvCmd(
				command{
					&protocol.Command{EntityId: "gcounter-1", Name: "GetGCounter"},
					&crdt3.Get{Key: "gcounter-1"},
				},
			)
			switch m := out.Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				err := encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				if err != nil {
					t.Fatal(err)
				}
				if got, want := value.GetValue(), uint64(21); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
				}
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
		t.Run("the counter should apply delta and return its value", func(t *testing.T) {
			err := p.sendDelta(delta{
				d: &protocol.CrdtDelta{Delta: &protocol.CrdtDelta_Gcounter{Gcounter: &protocol.GCounterDelta{Increment: uint64(7)}}},
			})
			if err != nil {
				t.Fatal(err)
			}

			out := p.sendRecvCmd(
				command{
					&protocol.Command{EntityId: "gcounter-1", Name: "GetGCounter"},
					&crdt3.Get{Key: "gcounter-1"},
				},
			)
			switch m := out.Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				err := encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				if err != nil {
					t.Fatal(err)
				}
				if got, want := value.GetValue(), uint64(28); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
				}
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
		t.Run("GetGCounter for a different entity id on the same connection should fail", func(t *testing.T) {
			out := p.sendRecvCmd(command{
				&protocol.Command{EntityId: "gcounter-2", Name: "GetGCounter"},
				&crdt3.Get{Key: "gcounter-2"},
			})
			switch m := out.Message.(type) {
			case *protocol.CrdtStreamOut_Failure:
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
	})
}
