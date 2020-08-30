package synth

import (
	"context"
	"testing"
	"time"

	"github.com/cloudstateio/go-support/cloudstate/protocol"
	crdt3 "github.com/cloudstateio/go-support/tck/proto/crdt"
)

func TestCRDTGCounter(t *testing.T) {
	s := newServer(t)
	s.newClientConn()
	defer s.teardown()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("GCounter", func(t *testing.T) {
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
			tr := tester{t}
			switch m := p.command(
				entityId, "IncrementGCounter", &crdt3.GCounterIncrement{Key: entityId, Value: 8},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				tr.expectedUInt64(value.GetValue(), 8)
				tr.expectedUInt64(m.Reply.GetStateAction().GetCreate().GetGcounter().GetValue(), 8)
			default:
				tr.unexpected(m)
			}
		})
		t.Run("a second increment should emit a client action and an update state action", func(t *testing.T) {
			tr := tester{t}
			switch m := p.command(
				entityId, "IncrementGCounter", &crdt3.GCounterIncrement{Key: entityId, Value: 8},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				tr.expectedUInt64(value.GetValue(), 16)
				tr.expectedUInt64(m.Reply.GetStateAction().GetUpdate().GetGcounter().GetIncrement(), 8)
			default:
				tr.unexpected(m)
			}
		})
		t.Run("calling GetGCounter should return the counters value", func(t *testing.T) {
			tr := tester{t}
			switch m := p.command(
				entityId, "GetGCounter", &crdt3.Get{Key: entityId},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				tr.expectedUInt64(value.GetValue(), 16)
			default:
				tr.unexpected(m)
			}
		})
		t.Run("the counter should apply new state and return its value", func(t *testing.T) {
			tr := tester{t}
			p.state(&protocol.GCounterState{Value: 24})
			switch m := p.command(
				entityId, "GetGCounter", &crdt3.Get{Key: entityId},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				tr.expectedUInt64(value.GetValue(), 24)
			default:
				tr.unexpected(m)
			}
		})
		t.Run("the counter should apply a delta and return its value", func(t *testing.T) {
			tr := tester{t}
			p.delta(&protocol.GCounterDelta{Increment: 8})
			switch m := p.command(
				entityId, "GetGCounter", &crdt3.Get{Key: entityId},
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
				tr.expectedUInt64(value.GetValue(), 32)
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

	t.Run("GCounter – unknown entity id used", func(t *testing.T) {
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
			tr := tester{t}
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

	t.Run("GCounter – incompatible CRDT delta sequence", func(t *testing.T) {
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

	t.Run("GCounter – incompatible CRDT delta used", func(t *testing.T) {
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
			tr := tester{t}
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

	t.Run("GCounter – inconsistent local state", func(t *testing.T) {
		entityId := "gcounter-3"
		tr := tester{t}
		p := newProxy(ctx, s)
		defer p.teardown()

		p.init(&protocol.CrdtInit{ServiceName: serviceName, EntityId: entityId})
		p.command(entityId, "IncrementGCounter", &crdt3.GCounterIncrement{Key: entityId, Value: 7})
		switch m := p.command(
			entityId, "IncrementGCounter", &crdt3.GCounterIncrement{
				Key: entityId, Value: 7, FailWith: "error",
			},
		).Message.(type) {
		case *protocol.CrdtStreamOut_Reply:
			switch a := m.Reply.GetClientAction().Action.(type) {
			case *protocol.ClientAction_Failure:
				tr.expectedString(a.Failure.GetDescription(), "error")
			default:
				tr.unexpected(a)
			}
		default:
			tr.unexpected(m)
		}

		// switch m := p.command(
		// 	entityId, "GetGCounter", &crdt3.Get{Key: entityId},
		// ).Message.(type) {
		// case *protocol.CrdtStreamOut_Reply:
		// 	value := crdt3.GCounterValue{}
		// 	tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
		// 	tr.expectedUInt64(value.GetValue(), uint64(14))
		// default:
		// 	tr.unexpected(m)
		// }

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
