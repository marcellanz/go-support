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
			one, err := encoding.Struct(pair{"one", 1})
			if err != nil {
				t.Fatal(err)
			}
			switch m := p.command(entityId, command, lwwRegisterRequest(&crdt.LWWRegisterSet{
				Value: &crdt.AnySupportType{
					Value: &crdt.AnySupportType_AnyValue{AnyValue: one},
				},
			}),
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				// action reply
				tr.expectedNil(m.Reply.GetSideEffects())
				tr.expectedNil(m.Reply.GetClientAction().GetFailure())
				var r crdt.LWWRegisterResponse
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &r)
				one, err := encoding.Struct(pair{"one", 1})
				if err != nil {
					t.Fatal(err)
				}
				tr.expectedOneIn([]*any.Any{r.GetValue().GetValue()}, one)
				// state action
				tr.expectedNotNil(m.Reply.GetStateAction().GetCreate())
				tr.expectedNil(m.Reply.GetStateAction().GetUpdate())
				tr.expectedNil(m.Reply.GetStateAction().GetDelete())
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
			two, err := encoding.Struct(pair{"two", 2})
			if err != nil {
				t.Fatal(err)
			}
			switch m := p.command(entityId, command, lwwRegisterRequest(&crdt.LWWRegisterSetWithClock{
				Value: &crdt.AnySupportType{
					Value: &crdt.AnySupportType_AnyValue{AnyValue: two},
				},
				Clock:            crdt.CrdtClock_CUSTOM,
				CustomClockValue: 7,
			}),
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				// action reply
				tr.expectedNil(m.Reply.GetSideEffects())
				tr.expectedNil(m.Reply.GetClientAction().GetFailure())
				var r crdt.LWWRegisterResponse
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &r)
				two, err := encoding.Struct(pair{"two", 2})
				if err != nil {
					t.Fatal(err)
				}
				tr.expectedOneIn([]*any.Any{r.GetValue().GetValue()}, two)
				// state action
				tr.expectedNil(m.Reply.GetStateAction().GetCreate())
				tr.expectedNotNil(m.Reply.GetStateAction().GetUpdate())
				tr.expectedNil(m.Reply.GetStateAction().GetDelete())
				var p pair
				tr.toStruct(m.Reply.GetStateAction().GetUpdate().GetLwwregister().GetValue(), &p)
				tr.expectedTrue(reflect.DeepEqual(p, pair{"two", 2}))
				tr.expectedTrue(m.Reply.GetStateAction().GetUpdate().GetLwwregister().GetClock() == protocol.CrdtClock_CUSTOM)
				tr.expectedTrue(m.Reply.GetStateAction().GetUpdate().GetLwwregister().GetCustomClockValue() == 7)
			default:
				tr.unexpected(m)
			}
		})
		t.Run("Delete emits client action and create state action", func(t *testing.T) {
			tr := tester{t}
			switch m := p.command(entityId, command, lwwRegisterRequest(&crdt.Delete{})).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				// action reply
				tr.expectedNil(m.Reply.GetSideEffects())
				tr.expectedNil(m.Reply.GetClientAction().GetFailure())
				var r crdt.LWWRegisterResponse
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &r)
				// state action
				tr.expectedNil(m.Reply.GetStateAction().GetCreate())
				tr.expectedNil(m.Reply.GetStateAction().GetUpdate())
				tr.expectedNotNil(m.Reply.GetStateAction().GetDelete())
			default:
				tr.unexpected(m)
			}
		})
	})
}
