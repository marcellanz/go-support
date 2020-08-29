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
	"strings"
	"testing"
	"time"

	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	crdt3 "github.com/cloudstateio/go-support/tck/proto/crdt"
)

// TestCRDT runs the TCK for the CRDT state model.
// As defined by the Cloudstate specification, each CRDT state model type
// has three state actions CRDTs can emit on state changes.
// - create
// - update
// - delete
func TestCRDT(t *testing.T) {
	s := newServer(t)
	s.newClientConn()
	defer s.teardown()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("entity discovery should find the service", func(t *testing.T) {
		edc := protocol.NewEntityDiscoveryClient(s.conn)
		discover, err := edc.Discover(ctx, &protocol.ProxyInfo{
			ProtocolMajorVersion: 0,
			ProtocolMinorVersion: 1,
			ProxyName:            "a-proxy",
			ProxyVersion:         "0.0.0",
			SupportedEntityTypes: []string{protocol.EventSourced, protocol.CRDT},
		})
		if err != nil {
			t.Fatal(err)
		}
		tr := tester{t}
		tr.expectedInt(len(discover.GetEntities()), 1)
		tr.expectedString(discover.GetEntities()[0].GetServiceName(), serviceName)
	})

	t.Run("GSet", func(t *testing.T) {
		entityId := "gset-1"
		p := newProxy(ctx, s)
		defer p.teardown()
		p.init(&protocol.CrdtInit{ServiceName: serviceName, EntityId: entityId})

		type pair struct {
			Left  string
			Right int64
		}
		t.Run("calling AddGSet should emit client and state action", func(t *testing.T) {
			tr := tester{t}
			one := &crdt3.AnySupportType{Value: &crdt3.AnySupportType_AnyValue{encoding.Struct(pair{"one", 1})}}

			switch m := p.command(
				entityId, "AddGSet", &crdt3.GSetAdd{Key: entityId, Value: one},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				// client action reply
				var reply crdt3.GSetValueAnySupport
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &reply)
				tr.expectedInt(len(reply.GetValues()), 1)

				var p pair
				if err := encoding.DecodeStruct(reply.GetValues()[0].GetAnyValue(), &p); err != nil {
					t.Fatal(err)
				}
				tr.expectedString(p.Left, "one")
				tr.expectedInt64(p.Right, 1)
				// state action
				i := m.Reply.GetStateAction().GetCreate().GetGset().GetItems()
				tr.expectedInt(len(i), 1)
				var state pair
				item := i[0]
				if err := encoding.DecodeStruct(item, &state); err != nil {
					t.Fatal(err)
				}

				tr.expectedBool(strings.HasPrefix(item.TypeUrl, encoding.JSONTypeURLPrefix), true)
				tr.expectedString(state.Left, "one")
				tr.expectedInt64(state.Right, 1)
			default:
				tr.unexpected(m)
			}
		})
		t.Run("adding more values should result in a larger set", func(t *testing.T) {
			tr := tester{t}
			switch m := p.command(
				entityId, "AddGSet",
				&crdt3.GSetAdd{Key: entityId,
					Value: &crdt3.AnySupportType{Value: &crdt3.AnySupportType_AnyValue{encoding.Struct(pair{"two", 3})}},
				},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				tr.expectedNotNil(m.Reply.GetStateAction().GetUpdate())
			default:
				tr.unexpected(m)
			}

			p.command(entityId, "AddGSet", &crdt3.GSetAdd{
				Key:   entityId,
				Value: &crdt3.AnySupportType{Value: &crdt3.AnySupportType_AnyValue{encoding.Struct(pair{"three", 3})}},
			})

			switch m := p.command(
				entityId, "GetGSetSize", &crdt3.Get{Key: entityId},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				var value crdt3.GSetSize
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				tr.expectedInt64(value.Value, 3)
			default:
				tr.unexpected(m)
			}
		})
	})

	t.Run("GSet AnySupportTypes", func(t *testing.T) {
		entityId := "gset-2"
		p := newProxy(ctx, s)
		defer p.teardown()
		p.init(&protocol.CrdtInit{
			ServiceName: serviceName,
			EntityId:    entityId,
		})

		values := []*crdt3.AnySupportType{
			{Value: &crdt3.AnySupportType_BoolValue{true}},
			{Value: &crdt3.AnySupportType_FloatValue{float32(1)}},
			{Value: &crdt3.AnySupportType_Int32Value{int32(2)}},
			{Value: &crdt3.AnySupportType_Int64Value{int64(3)}},
			{Value: &crdt3.AnySupportType_DoubleValue{float64(4.4)}},
			{Value: &crdt3.AnySupportType_StringValue{"five"}},
			{Value: &crdt3.AnySupportType_BytesValue{[]byte{'a', 'b', 3, 4, 5, 6}}},
		}
		p.command(entityId, "AddGSet", &crdt3.GSetAdd{Key: entityId, Value: values[0]})
	})

	t.Run("PNCounter", func(t *testing.T) {
		entityId := "pncounter-1"
		p := newProxy(ctx, s)
		defer p.teardown()
		p.init(&protocol.CrdtInit{
			ServiceName: serviceName,
			EntityId:    entityId,
		})
		tr := tester{t}
		t.Run("incrementing a PNCounter should emit client action and create-state action", func(t *testing.T) {
			switch m := p.command(
				entityId, "IncrementPNCounter", &crdt3.PNCounterIncrement{Key: entityId, Value: 7},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				var value crdt3.PNCounterValue
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				tr.expectedInt64(value.GetValue(), 7)
				tr.expectedInt64(m.Reply.GetStateAction().GetCreate().GetPncounter().GetValue(), 7)
			default:
				tr.unexpected(m)
			}
		})
		t.Run("a second increment should emit a client action and an update state action", func(t *testing.T) {
			switch m := p.command(
				entityId, "IncrementPNCounter", &crdt3.PNCounterIncrement{Key: entityId, Value: 7},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				var value crdt3.PNCounterValue
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				tr.expectedInt64(value.GetValue(), 14)
				tr.expectedInt64(m.Reply.GetStateAction().GetUpdate().GetPncounter().GetChange(), 7)
			default:
				tr.unexpected(m)
			}
		})
		t.Run("a decrement should emit a client action and an update state action", func(t *testing.T) {
			switch m := p.command(
				entityId, "DecrementPNCounter", &crdt3.PNCounterDecrement{Key: entityId, Value: 22},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				var value crdt3.PNCounterValue
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				tr.expectedInt64(value.GetValue(), -8)
				tr.expectedInt64(m.Reply.GetStateAction().GetUpdate().GetPncounter().GetChange(), -22)
			default:
				tr.unexpected(m)
			}
		})
		t.Run("the counter should apply new state and return its value", func(t *testing.T) {
			p.state(&protocol.PNCounterState{Value: int64(49)})
			switch m := p.command(
				entityId, "GetPNCounter", &crdt3.Get{Key: entityId},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				var value crdt3.PNCounterValue
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				tr.expectedInt64(value.GetValue(), int64(49))
				tr.expectedNil(m.Reply.GetClientAction().GetFailure())
			default:
				tr.unexpected(m)
			}
		})
		t.Run("the counter should apply a delta and return its value", func(t *testing.T) {
			p.delta(&protocol.PNCounterDelta{Change: int64(-52)})
			switch m := p.command(
				entityId, "GetPNCounter", &crdt3.Get{Key: entityId},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				var value crdt3.PNCounterValue
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				tr.expectedInt64(value.GetValue(), int64(-3))
			default:
				tr.unexpected(m)
			}
		})
	})

	t.Run("GCounter", func(t *testing.T) {
		tr := tester{t}
		p := newProxy(ctx, s)
		defer p.teardown()
		entityId := "gcounter-0"
		t.Run("send a CrdtInit should not fail", func(t *testing.T) {
			p.init(&protocol.CrdtInit{
				ServiceName: serviceName,
				EntityId:    entityId,
			})
		})
		t.Run("incrementing a GCounter should emit a client action and create state action", func(t *testing.T) {
			switch m := p.command(
				entityId, "IncrementGCounter", &crdt3.GCounterIncrement{Key: entityId, Value: 7},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				tr.expectedUInt64(value.GetValue(), 7)
				tr.expectedUInt64(m.Reply.GetStateAction().GetCreate().GetGcounter().GetValue(), uint64(7))
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
		t.Run("a second increment should emit a client action and an update state action", func(t *testing.T) {
			switch m := p.command(
				entityId, "IncrementGCounter", &crdt3.GCounterIncrement{Key: entityId, Value: 7},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				tr.expectedUInt64(value.GetValue(), uint64(14))
				tr.expectedUInt64(m.Reply.GetStateAction().GetUpdate().GetGcounter().GetIncrement(), uint64(7))
			default:
				tr.unexpected(m)
			}
		})
		t.Run("calling GetGCounter should return the counters value", func(t *testing.T) {
			switch m := p.command(
				entityId, "GetGCounter", &crdt3.Get{Key: entityId},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				tr.expectedUInt64(value.GetValue(), uint64(14))
			default:
				tr.unexpected(m)
			}
		})
		t.Run("the counter should apply new state and return its value", func(t *testing.T) {
			p.state(&protocol.GCounterState{Value: uint64(21)})
			switch m := p.command(
				entityId, "GetGCounter", &crdt3.Get{Key: entityId},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				tr.expectedUInt64(value.GetValue(), uint64(21))
			default:
				tr.unexpected(m)
			}
		})
		t.Run("the counter should apply a delta and return its value", func(t *testing.T) {
			p.delta(&protocol.GCounterDelta{Increment: uint64(7)})
			switch m := p.command(
				entityId, "GetGCounter", &crdt3.Get{Key: entityId},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				tr.expectedUInt64(value.GetValue(), uint64(28))
			default:
				tr.unexpected(m)
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
			p.init(&protocol.CrdtInit{
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
	})

	t.Run("GCounter with unknown entity id used", func(t *testing.T) {
		tr := tester{t}
		p := newProxy(ctx, s)
		defer p.teardown()
		entityId := "gcounter-1"
		p.init(&protocol.CrdtInit{
			ServiceName: serviceName,
			EntityId:    entityId,
		})

		switch m := p.command(
			entityId, "IncrementGCounter", &crdt3.GCounterIncrement{Key: entityId, Value: 8},
		).Message.(type) {
		case *protocol.CrdtStreamOut_Failure:
			tr.unexpected(m)
		}
		t.Run("calling GetGCounter for a non existing entity id should fail", func(t *testing.T) {
			entityId := "gcounter-1-xxx"
			switch m := p.command(
				entityId, "GetGCounter", &crdt3.Get{Key: entityId},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Failure:
			default:
				tr.unexpected(m)
			}
		})
	})

	t.Run("GCounter with incompatible CRDT delta sequence", func(t *testing.T) {
		p := newProxy(ctx, s)
		defer p.teardown()
		entityId := "gcounter-2"
		p.init(&protocol.CrdtInit{
			ServiceName: serviceName,
			EntityId:    entityId,
		})
		t.Run("setting a delta without ever sending state should fail", func(t *testing.T) {
			t.Skip("we can't test this one for now")
			p.delta(&protocol.GCounterDelta{Increment: 7})
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
		tr := tester{t}
		p := newProxy(ctx, s)
		defer p.teardown()
		entityId := "gcounter-2"

		p.init(&protocol.CrdtInit{
			ServiceName: serviceName,
			EntityId:    entityId,
		})
		switch m := p.command(
			entityId, "IncrementGCounter", &crdt3.GCounterIncrement{Key: entityId, Value: 8},
		).Message.(type) {
		case *protocol.CrdtStreamOut_Failure:
			tr.unexpected(m)
		}
		t.Run("setting a delta for a different CRDT type should fail", func(t *testing.T) {
			p.delta(&protocol.PNCounterDelta{Change: 7})
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
				tr.unexpected(m)
			}
		})
	})

	t.Run("GCounter with inconsistent local state", func(t *testing.T) {
		entityId := "gcounter-3"
		tstr := tester{t}
		p := newProxy(ctx, s)
		defer p.teardown()

		p.init(&protocol.CrdtInit{ServiceName: serviceName, EntityId: entityId})
		p.command(entityId, "IncrementGCounter", &crdt3.GCounterIncrement{Key: entityId, Value: 42})
		switch m := p.command(
			entityId, "IncrementGCounter", &crdt3.GCounterIncrement{
				Key: entityId, Value: 52, FailWith: "error",
			},
		).Message.(type) {
		case *protocol.CrdtStreamOut_Reply:
			switch a := m.Reply.GetClientAction().Action.(type) {
			case *protocol.ClientAction_Failure:
				tstr.expectedString(a.Failure.GetDescription(), "error")
			default:
				tstr.unexpected(a)
			}
		default:
			tstr.unexpected(m)
		}

		// TODO: revise this tests as we should have a stream still up after a client failure
		// _, err :=
		// if err == nil {
		//	t.Fatal("expected err")
		// }
		// if err != io.EOF {
		//	t.Fatal("expected io.EOF")
		// }
		//
		// switch m := p.sendCmdRecvReply(command{
		//	&protocol.Command{EntityId: entityId, Name: "IncrementGCounter"},
		//	&crdt3.GCounterIncrement{Key: entityId, Value: 9},
		// }).Message.(type) {
		// case *protocol.CrdtStreamOut_Reply:
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
		// default:
		//	t.Fatalf("got unexpected message: %+v", m)
		// }
	})
}
