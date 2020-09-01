package synth

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/cloudstateio/go-support/tck/proto/crdt"
	"github.com/golang/protobuf/ptypes/any"
)

func TestCRDTLWWRegister(t *testing.T) {
	s := newServer(t)
	defer s.teardown()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	type pair struct {
		Left  string
		Right int64
	}
	t.Run("LWWRegister", func(t *testing.T) {
		entityId := "lwwregister-1"
		command := "ProcessORSet"
		p := newProxy(ctx, s)
		defer p.teardown()

		p.init(&protocol.CrdtInit{ServiceName: serviceName, EntityId: entityId})
		t.Run("LWWRegisterSet emits client action and create state action", func(t *testing.T) {
			tr := tester{t}
			switch m := p.command(entityId, command, lwwRegisterRequest(&crdt.LWWRegisterSet{
				Value: &crdt.AnySupportType{
					Value: &crdt.AnySupportType_AnyValue{AnyValue: encoding.Struct(pair{"one", 1})},
				},
			}),
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				// action reply
				tr.expectedNil(m.Reply.GetSideEffects())
				tr.expectedNil(m.Reply.GetClientAction().GetFailure())
				var reg crdt.LWWRegisterValue
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &reg)
				tr.expectedOneIn([]*any.Any{reg.GetValue()}, encoding.Struct(pair{"one", 1}))
				// state
				tr.expectedNotNil(m.Reply.GetStateAction().GetCreate())
				var p pair
				tr.expectedTrue(m.Reply.GetStateAction().GetCreate().GetLwwregister().GetClock() == protocol.CrdtClock_DEFAULT)
				tr.toStruct(m.Reply.GetStateAction().GetCreate().GetLwwregister().GetValue(), &p)
				tr.expectedTrue(reflect.DeepEqual(p, pair{"one", 1}))
			default:
				tr.unexpected(m)
			}
		})
		t.Run("LWWRegisterSetWithClock emits client action and update state action", func(t *testing.T) {
			tr := tester{t}
			switch m := p.command(entityId, command, lwwRegisterRequest(&crdt.LWWRegisterSetWithClock{
				Value: &crdt.AnySupportType{
					Value: &crdt.AnySupportType_AnyValue{AnyValue: encoding.Struct(pair{"two", 2})},
				},
				Clock:            crdt.CrdtClock_CUSTOM,
				CustomClockValue: 7,
			}),
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				// action reply
				tr.expectedNil(m.Reply.GetSideEffects())
				tr.expectedNil(m.Reply.GetClientAction().GetFailure())
				var reg crdt.LWWRegisterValue
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &reg)
				tr.expectedOneIn([]*any.Any{reg.GetValue()}, encoding.Struct(pair{"two", 2}))
				// state
				tr.expectedNil(m.Reply.GetStateAction().GetCreate())
				var p pair
				tr.toStruct(m.Reply.GetStateAction().GetUpdate().GetLwwregister().GetValue(), &p)
				tr.expectedTrue(reflect.DeepEqual(p, pair{"two", 2}))
				tr.expectedTrue(m.Reply.GetStateAction().GetUpdate().GetLwwregister().GetClock() == protocol.CrdtClock_CUSTOM)
				tr.expectedTrue(m.Reply.GetStateAction().GetUpdate().GetLwwregister().GetCustomClockValue() == 7)
			default:
				tr.unexpected(m)
			}
		})
	})
}
