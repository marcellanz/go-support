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

	t.Run("entity discovery should find the service", func(t *testing.T) {
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

	t.Run("PNCounter", func(t *testing.T) {
		entityId := "pncounter-1"
		p := newProxy(ctx, s)
		defer p.teardown()
		p.sendInit(&protocol.CrdtInit{
			ServiceName: serviceName,
			EntityId:    entityId,
		})
		t.Run("incrementing a PNCounter should emit a client action and create state action", func(t *testing.T) {
			switch m := p.sendCmdRecvReply(command{
				&protocol.Command{EntityId: entityId, Name: "IncrementPNCounter"},
				&crdt3.PNCounterIncrement{Key: entityId, Value: 7},
			}).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				var value crdt3.PNCounterValue
				if err := encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), &value); err != nil {
					t.Fatal(err)
				}
				if got, want := value.GetValue(), int64(7); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
				}
				if got, want := m.Reply.GetStateAction().GetCreate().GetPncounter().GetValue(), int64(7); got != want {
					t.Fatalf("got = %v; wanted: %d", got, want)
				}
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
			//switch m := p.sendCmdRecvReply(command{
			//	&protocol.Command{EntityId: entityId, Name: "GetPNCounter"},
			//	&crdt3.Get{Key: entityId},
			//}).Message.(type) {
			//case *protocol.CrdtStreamOut_Reply:
			//	switch a := m.Reply.GetClientAction().Action.(type) {
			//	case *protocol.ClientAction_Reply:
			//		var value crdt3.PNCounterValue
			//		if err := encoding.UnmarshalAny(a.Reply.Payload, &value); err != nil {
			//			t.Fatal(err)
			//		}
			//		if got, want := value.GetValue(), int64(7); got != want {
			//			t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
			//		}
			//	}
			//default:
			//	t.Fatalf("got unexpected message: %+v", m)
			//}
		})
		t.Run("a second increment should emit a client action and an update state action", func(t *testing.T) {
			switch m := p.sendCmdRecvReply(command{
				&protocol.Command{EntityId: entityId, Name: "IncrementPNCounter"},
				&crdt3.PNCounterIncrement{Key: entityId, Value: 7},
			}).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				var value crdt3.PNCounterValue
				if err := encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), &value); err != nil {
					t.Fatal(err)
				}
				if got, want := value.GetValue(), int64(14); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
				}
				if got, want := m.Reply.GetStateAction().GetUpdate().GetPncounter().GetChange(), int64(7); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
				}
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
		t.Run("a decrement should emit a client action and an update state action", func(t *testing.T) {
			switch m := p.sendCmdRecvReply(command{
				&protocol.Command{EntityId: entityId, Name: "DecrementPNCounter"},
				&crdt3.PNCounterDecrement{Key: entityId, Value: 22},
			}).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				var value crdt3.PNCounterValue
				if err := encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), &value); err != nil {
					t.Fatal(err)
				}
				if got, want := value.GetValue(), int64(-8); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
				}
				if got, want := m.Reply.GetStateAction().GetUpdate().GetPncounter().GetChange(), int64(-22); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
				}
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
		t.Run("the counter should apply new state and return its value", func(t *testing.T) {
			err := p.sendState(state{
				&protocol.CrdtState{State: &protocol.CrdtState_Pncounter{
					Pncounter: &protocol.PNCounterState{Value: int64(49)},
				}},
			})
			if err != nil {
				t.Fatal(err)
			}
			switch m := p.sendCmdRecvReply(command{
				&protocol.Command{EntityId: entityId, Name: "GetPNCounter"},
				&crdt3.Get{Key: entityId},
			}).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				var value crdt3.PNCounterValue
				if err := encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), &value); err != nil {
					t.Fatal(err)
				}
				if got, want := value.GetValue(), int64(49); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
				}
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
		t.Run("the counter should apply a delta and return its value", func(t *testing.T) {
			p.sendDelta(delta{
				&protocol.CrdtDelta{Delta: &protocol.CrdtDelta_Pncounter{
					Pncounter: &protocol.PNCounterDelta{Change: int64(-52)}},
				},
			})
			switch m := p.sendCmdRecvReply(command{
				&protocol.Command{EntityId: entityId, Name: "GetPNCounter"},
				&crdt3.Get{Key: entityId},
			}).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				var value crdt3.PNCounterValue
				if err := encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), &value); err != nil {
					t.Fatal(err)
				}
				if got, want := value.GetValue(), int64(-3); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
				}
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
	})

	t.Run("GCounter", func(t *testing.T) {
		p := newProxy(ctx, s)
		defer p.teardown()
		entityId := "gcounter-0"
		t.Run("send a CrdtInit should not fail", func(t *testing.T) {
			p.sendInit(&protocol.CrdtInit{
				ServiceName: serviceName,
				EntityId:    entityId,
			})
		})
		t.Run("incrementing a GCounter should emit a client action and create state action", func(t *testing.T) {
			out := p.sendCmdRecvReply(command{
				&protocol.Command{EntityId: entityId, Name: "IncrementGCounter"},
				&crdt3.GCounterIncrement{Key: entityId, Value: 7},
			})
			switch m := out.Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				err := encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				if err != nil {
					t.Fatal(err)
				}
				if got, want := value.GetValue(), uint64(7); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
				}
				if got, want := m.Reply.GetStateAction().GetCreate().GetGcounter().GetValue(), uint64(7); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, out.GetReply())
				}
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
		t.Run("a second increment should emit a client action and an update state action", func(t *testing.T) {
			out := p.sendCmdRecvReply(command{
				&protocol.Command{EntityId: entityId, Name: "IncrementGCounter"},
				&crdt3.GCounterIncrement{Key: entityId, Value: 7},
			})
			switch m := out.Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				if err := encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), &value); err != nil {
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
			out := p.sendCmdRecvReply(
				command{
					&protocol.Command{EntityId: entityId, Name: "GetGCounter"},
					&crdt3.Get{Key: entityId},
				},
			)
			switch m := out.Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				if err := encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), &value); err != nil {
					t.Fatal(err)
				}
				if got, want := value.GetValue(), uint64(14); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
				}
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
		t.Run("the counter should apply new state and return its value", func(t *testing.T) {
			err := p.sendState(state{
				&protocol.CrdtState{State: &protocol.CrdtState_Gcounter{
					Gcounter: &protocol.GCounterState{
						Value: uint64(21)},
				}},
			})
			if err != nil {
				t.Fatal(err)
			}
			out := p.sendCmdRecvReply(command{
				&protocol.Command{EntityId: entityId, Name: "GetGCounter"},
				&crdt3.Get{Key: entityId},
			})
			switch m := out.Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				if err := encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), &value); err != nil {
					t.Fatal(err)
				}
				if got, want := value.GetValue(), uint64(21); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
				}
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
		t.Run("the counter should apply a delta and return its value", func(t *testing.T) {
			p.sendDelta(delta{
				d: &protocol.CrdtDelta{Delta: &protocol.CrdtDelta_Gcounter{Gcounter: &protocol.GCounterDelta{Increment: uint64(7)}}},
			})
			out := p.sendCmdRecvReply(command{
				&protocol.Command{EntityId: entityId, Name: "GetGCounter"},
				&crdt3.Get{Key: entityId},
			})
			switch m := out.Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				if err := encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), &value); err != nil {
					t.Fatal(err)
				}
				if got, want := value.GetValue(), uint64(28); got != want {
					t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
				}
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
		t.Run("deleting an entity should delete the entity", func(t *testing.T) {
			p.sendDelete(delete{d: &protocol.CrdtDelete{}})
		})
		t.Run("after an entity was deleted, we could initialise an another entity", func(t *testing.T) {
			// this is not explicit specified by the spec, but it says, that the user function should
			// clear its state and the proxy could close the stream anytime, but also does not say
			// the user function can close the stream. So our implementation would be prepared for a
			// new entity re-using the same stream (why not).
			p.sendInit(&protocol.CrdtInit{
				ServiceName: serviceName,
				EntityId:    "gcounter-xyz",
			})
			// nothing should be returned here
			resp, err := p.Recv()
			if err != nil {
				t.Fatal(err)
			}
			if resp != nil {
				t.Fatal("no response expected")
			}
		})

		//recv, err := p.recv()
		//if err != nil {
		//	t.Fatal(err)
		//}
		//switch msg := recv.Message.(type) {
		//case *protocol.CrdtStreamOut_Failure:
		//	t.Fatal(msg.Failure.Description)
		//default:
		//	t.Fatalf("got unexpected message: %+v", msg)
		//}

		//recvC := make(chan resp, 1)
		//go func() {
		//	recv, err := p.h.Recv()
		//	recvC <- resp{recv, err}
		//}()
		//select {
		//case <-time.After(1 * time.Second):
		//	t.Log("no reponse")
		//case m := <-recvC:
		//	if m.err != nil {
		//		t.Fatal(m.err)
		//	}
		//	switch msg := m.msg.Message.(type) {
		//	case *protocol.CrdtStreamOut_Failure:
		//		t.Fatal(msg.Failure.Description)
		//	default:
		//		t.Fatalf("got unexpected message: %+v", m)
		//	}
		//}
	})

	t.Run("GCounter with unknown entity id used", func(t *testing.T) {
		p := newProxy(ctx, s)
		defer p.teardown()
		entityId := "gcounter-1"
		p.sendInit(&protocol.CrdtInit{
			ServiceName: serviceName,
			EntityId:    entityId,
		})
		switch m := p.sendCmdRecvReply(command{
			&protocol.Command{EntityId: entityId, Name: "IncrementGCounter"},
			&crdt3.GCounterIncrement{Key: entityId, Value: 8},
		}).Message.(type) {
		case *protocol.CrdtStreamOut_Failure:
			t.Fatalf("got unexpected message: %+v", m)
		}
		t.Run("calling GetGCounter for a non existing entity id should fail", func(t *testing.T) {
			entityId := "gcounter-1-xxx"
			out := p.sendCmdRecvReply(command{
				&protocol.Command{EntityId: entityId, Name: "GetGCounter"},
				&crdt3.Get{Key: entityId},
			})
			switch m := out.Message.(type) {
			case *protocol.CrdtStreamOut_Failure:
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
	})

	t.Run("GCounter with incompatible CRDT delta sequence", func(t *testing.T) {
		p := newProxy(ctx, s)
		defer p.teardown()
		entityId := "gcounter-2"
		p.sendInit(&protocol.CrdtInit{
			ServiceName: serviceName,
			EntityId:    entityId,
		})
		//switch m := p.sendCmdRecvReply(command{
		//	&protocol.Command{EntityId: entityId, Name: "IncrementGCounter"},
		//	&crdt3.GCounterIncrement{Key: entityId, Value: 8},
		//}).Message.(type) {
		//case *protocol.CrdtStreamOut_Failure:
		//	t.Fatalf("got unexpected message: %+v", m)
		//}
		t.Run("setting a delta without ever sending state should fail", func(t *testing.T) {
			p.sendDelta(delta{
				d: &protocol.CrdtDelta{Delta: &protocol.CrdtDelta_Gcounter{Gcounter: &protocol.GCounterDelta{
					Increment: 7,
				}}},
			})
			// nothing should be returned here
			resp, err := p.Recv()
			if err != nil {
				t.Fatal(err)
			}
			if resp == nil {
				t.Fatal("response expected")
			}
			switch m := resp.Message.(type) {
			case *protocol.CrdtStreamOut_Failure:
				// the expected failure
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
	})

	t.Run("GCounter with incompatible CRDT delta used", func(t *testing.T) {
		p := newProxy(ctx, s)
		defer p.teardown()
		entityId := "gcounter-2"
		p.sendInit(&protocol.CrdtInit{
			ServiceName: serviceName,
			EntityId:    entityId,
		})
		switch m := p.sendCmdRecvReply(command{
			&protocol.Command{EntityId: entityId, Name: "IncrementGCounter"},
			&crdt3.GCounterIncrement{Key: entityId, Value: 8},
		}).Message.(type) {
		case *protocol.CrdtStreamOut_Failure:
			t.Fatalf("got unexpected message: %+v", m)
		}
		t.Run("setting a delta for a different CRDT type should fail", func(t *testing.T) {
			p.sendDelta(delta{
				&protocol.CrdtDelta{Delta: &protocol.CrdtDelta_Pncounter{Pncounter: &protocol.PNCounterDelta{
					Change: 7,
				}}},
			})
			// nothing should be returned here
			resp, err := p.Recv()
			if err != nil {
				t.Fatal(err)
			}
			if resp == nil {
				t.Fatal("response expected")
			}
			switch m := resp.Message.(type) {
			case *protocol.CrdtStreamOut_Failure:
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
	})

	t.Run("GCounter with inconsistent local state", func(t *testing.T) {
		entityId := "gcounter-3"
		p := newProxy(ctx, s)
		defer p.teardown()
		p.sendInit(&protocol.CrdtInit{
			ServiceName: serviceName,
			EntityId:    entityId,
		})
		switch m := p.sendCmdRecvReply(command{
			&protocol.Command{EntityId: entityId, Name: "IncrementGCounter"},
			&crdt3.GCounterIncrement{Key: entityId, Value: 8},
		}).Message.(type) {
		case *protocol.CrdtStreamOut_Failure:
			t.Fatalf("got unexpected message: %+v", m)
		}
		switch m := p.sendCmdRecvReply(command{
			&protocol.Command{EntityId: entityId, Name: "IncrementGCounter"},
			&crdt3.GCounterIncrement{Key: entityId, Value: 1000},
		}).Message.(type) {
		case *protocol.CrdtStreamOut_Reply:
			switch m.Reply.GetClientAction().Action.(type) {
			case *protocol.ClientAction_Failure:
				// expected to fail with a failure
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		default:
			t.Fatalf("got unexpected message: %+v", m)
		}

		// TODO: revise this tests as we should have a stream still up after a client failure
		//_, err :=
		//if err == nil {
		//	t.Fatal("expected err")
		//}
		//if err != io.EOF {
		//	t.Fatal("expected io.EOF")
		//}
		//
		//switch m := p.sendCmdRecvReply(command{
		//	&protocol.Command{EntityId: entityId, Name: "IncrementGCounter"},
		//	&crdt3.GCounterIncrement{Key: entityId, Value: 9},
		//}).Message.(type) {
		//case *protocol.CrdtStreamOut_Reply:
		//	value := crdt3.GCounterValue{}
		//	err := encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
		//	if err != nil {
		//		t.Fatal(err)
		//	}
		//	if got, want := value.GetValue(), uint64(8+9); got != want {
		//		t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
		//	}
		//	if got, want := m.Reply.GetStateAction().GetUpdate().GetGcounter().GetIncrement(), uint64(9); got != want {
		//		t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, m.Reply)
		//	}
		//default:
		//	t.Fatalf("got unexpected message: %+v", m)
		//}
	})
}
