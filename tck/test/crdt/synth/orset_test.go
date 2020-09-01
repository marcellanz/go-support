package synth

import (
	"context"
	"testing"
	"time"

	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/cloudstateio/go-support/tck/proto/crdt"
	"github.com/golang/protobuf/ptypes/empty"
)

func TestCRDTORSet(t *testing.T) {
	s := newServer(t)
	s.newClientConn()
	defer s.teardown()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	type pair struct {
		Left  string
		Right int64
	}
	t.Run("ORSet", func(t *testing.T) {
		entityId := "orset-1"
		command := "ProcessORSet"
		p := newProxy(ctx, s)
		defer p.teardown()

		p.init(&protocol.CrdtInit{ServiceName: serviceName, EntityId: entityId})
		t.Run("ORSetAdd emits client action and create state action", func(t *testing.T) {
			tr := tester{t}
			switch m := p.command(entityId, command, orsetRequest(&crdt.ORSetAdd{Value: &crdt.AnySupportType{
				Value: &crdt.AnySupportType_AnyValue{AnyValue: encoding.Struct(pair{"one", 1})}},
			}),
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				// action reply
				tr.expectedNil(m.Reply.GetSideEffects())
				tr.expectedNil(m.Reply.GetClientAction().GetFailure())
				var set crdt.ORSetValue
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &set)
				tr.expectedOneIn(set.GetValues(), encoding.Struct(pair{"one", 1}))
				// state
				tr.expectedNotNil(m.Reply.GetStateAction().GetCreate())
				tr.expectedOneIn(
					m.Reply.GetStateAction().GetCreate().GetOrset().GetItems(),
					encoding.Struct(pair{"one", 1}),
				)
			default:
				tr.unexpected(m)
			}
		})
		t.Run("ORSetRemove emits client action and update state action", func(t *testing.T) {
			tr := tester{t}
			p.command(entityId, command, orsetRequest(&crdt.ORSetAdd{Value: &crdt.AnySupportType{
				Value: &crdt.AnySupportType_AnyValue{AnyValue: encoding.Struct(pair{"two", 2})}},
			}))
			switch m := p.command(entityId, command, orsetRequest(&crdt.ORSetRemove{Value: &crdt.AnySupportType{
				Value: &crdt.AnySupportType_AnyValue{AnyValue: encoding.Struct(pair{"one", 1})}},
			}),
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				// action reply
				tr.expectedNil(m.Reply.GetSideEffects())
				tr.expectedNil(m.Reply.GetClientAction().GetFailure())
				tr.expectedNil(m.Reply.GetClientAction().GetForward())
				var set crdt.ORSetValue
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &set)
				tr.expectedOneIn(set.GetValues(), encoding.Struct(pair{"two", 2}))
				tr.expectedInt(len(set.GetValues()), 1)
				// state
				tr.expectedNotNil(m.Reply.GetStateAction().GetUpdate())
				tr.expectedOneIn(
					m.Reply.GetStateAction().GetUpdate().GetOrset().GetRemoved(),
					encoding.Struct(pair{"one", 1}),
				)
			default:
				tr.unexpected(m)
			}
		})
		t.Run("ORSetRemove emits client action and update state action", func(t *testing.T) {
			tr := tester{t}
			p.command(entityId, command, orsetRequest(&crdt.ORSetAdd{Value: &crdt.AnySupportType{
				Value: &crdt.AnySupportType_AnyValue{AnyValue: encoding.Struct(pair{"two", 2})}},
			}))
			switch m := p.command(
				entityId, command, orsetRequest(&crdt.Delete{}),
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				tr.expectedNil(m.Reply.GetSideEffects())
				e := &empty.Empty{}
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), e)
				tr.expectedNotNil(m.Reply.GetStateAction().GetDelete())
			default:
				tr.unexpected(m)
			}
		})
	})
}
