package synth

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/cloudstateio/go-support/tck/proto/crdt"
)

func TestCRDTGSet(t *testing.T) {
	s := newServer(t)
	s.newClientConn()
	defer s.teardown()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("GSet", func(t *testing.T) {
		entityId := "gset-1"
		command := "ProcessGSet"
		p := newProxy(ctx, s)
		defer p.teardown()
		p.init(&protocol.CrdtInit{ServiceName: serviceName, EntityId: entityId})

		type pair struct {
			Left  string
			Right int64
		}
		t.Run("calling AddGSet should emit client action and create state action", func(t *testing.T) {
			tr := tester{t}
			switch m := p.command(
				entityId, command, gsetRequest(&crdt.GSetAdd{Key: entityId, Value: &crdt.AnySupportType{Value: &crdt.AnySupportType_AnyValue{AnyValue: encoding.Struct(pair{"one", 1})}}}),
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				tr.expectedNil(m.Reply.GetStateAction().GetUpdate())
				tr.expectedNil(m.Reply.GetStateAction().GetDelete())
				// action reply
				var r crdt.GSetResponse
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &r)
				tr.expectedInt(len(r.GetValue().GetValues()), 1)
				var p pair
				tr.toStruct(r.GetValue().GetValues()[0].GetAnyValue(), &p)
				tr.expectedString(p.Left, "one")
				tr.expectedInt64(p.Right, 1)
				// create state action
				tr.expectedInt(len(m.Reply.GetStateAction().GetCreate().GetGset().GetItems()), 1)
				i := m.Reply.GetStateAction().GetCreate().GetGset().GetItems()[0]
				tr.expectedBool(strings.HasPrefix(i.TypeUrl, encoding.JSONTypeURLPrefix), true)
				var state pair
				tr.toStruct(i, &state)
				tr.expectedString(state.Left, "one")
				tr.expectedInt64(state.Right, 1)
			default:
				tr.unexpected(m)
			}
		})
		t.Run("further calls of AddGSet should emit client action and delta state action", func(t *testing.T) {
			tr := tester{t}
			switch m := p.command(
				entityId, command, gsetRequest(&crdt.GSetAdd{Key: entityId, Value: &crdt.AnySupportType{Value: &crdt.AnySupportType_AnyValue{AnyValue: encoding.Struct(pair{"two", 2})}}}),
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				tr.expectedNil(m.Reply.GetStateAction().GetCreate())
				tr.expectedNil(m.Reply.GetStateAction().GetDelete())
				// action reply
				var r crdt.GSetResponse
				tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &r)
				tr.expectedInt(len(r.GetValue().GetValues()), 2)
				found := 0
				for _, v := range r.GetValue().GetValues() {
					var p pair
					tr.toStruct(v.GetAnyValue(), &p)
					if reflect.DeepEqual(p, pair{Left: "one", Right: 1}) {
						found++
					}
					if reflect.DeepEqual(p, pair{Left: "two", Right: 2}) {
						found++
					}
				}
				tr.expectedInt(found, 2)
				// update state action
				tr.expectedInt(len(m.Reply.GetStateAction().GetUpdate().GetGset().GetAdded()), 1)
				i := m.Reply.GetStateAction().GetUpdate().GetGset().GetAdded()[0]
				tr.expectedBool(strings.HasPrefix(i.TypeUrl, encoding.JSONTypeURLPrefix), true)
				var state pair
				tr.toStruct(i, &state)
				tr.expectedString(state.Left, "two")
				tr.expectedInt64(state.Right, 2)
			default:
				tr.unexpected(m)
			}
		})
		// t.Run("further calls of AddGSet should emit client action and delta state action", func(t *testing.T) {
		// 	tr := tester{t}
		// }
		// t.Run("adding more values should result in a larger set", func(t *testing.T) {
		// 	tr := tester{t}
		// 	// for pr := range []pair{
		// 	// 	{"two", 2},
		// 	// 	{"three", 3},
		// 	// } {
		// 	// 	switch m := p.command(
		// 	// 		entityId, "AddGSet",
		// 	// 		&crdt.GSetAdd{Key: entityId,
		// 	// 			Value: &crdt.AnySupportType{Value: &crdt.AnySupportType_AnyValue{AnyValue: encoding.Struct(pr)}},
		// 	// 		},
		// 	// 	).Message.(type) {
		// 	// 	case *protocol.CrdtStreamOut_Reply:
		// 	// 		tr.expectedNotNil(m.Reply.GetStateAction().GetUpdate())
		// 	// 	default:
		// 	// 		tr.unexpected(m)
		// 	// 	}
		// 	// }
		// 	// switch m := p.command(
		// 	// 	entityId, "GetGSetSize", &crdt.Get{Key: entityId},
		// 	// ).Message.(type) {
		// 	// case *protocol.CrdtStreamOut_Reply:
		// 	// 	var value crdt.GSetSize
		// 	// 	tr.toProto(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
		// 	// 	tr.expectedInt64(value.Value, 3)
		// 	// default:
		// 	// 	tr.unexpected(m)
		// 	// }
		// })
	})

	t.Run("GSet AnySupportTypes", func(t *testing.T) {
		entityId := "gset-2"
		command := "ProcessGSet"
		p := newProxy(ctx, s)
		defer p.teardown()
		p.init(&protocol.CrdtInit{
			ServiceName: serviceName,
			EntityId:    entityId,
		})

		var values = []*crdt.AnySupportType{
			{Value: &crdt.AnySupportType_BoolValue{BoolValue: true}},
			{Value: &crdt.AnySupportType_FloatValue{FloatValue: float32(1)}},
			{Value: &crdt.AnySupportType_Int32Value{Int32Value: int32(2)}},
			{Value: &crdt.AnySupportType_Int64Value{Int64Value: int64(3)}},
			{Value: &crdt.AnySupportType_DoubleValue{DoubleValue: 4.4}},
			{Value: &crdt.AnySupportType_StringValue{StringValue: "five"}},
			{Value: &crdt.AnySupportType_BytesValue{BytesValue: []byte{'a', 'b', 3, 4, 5, 6}}},
		}
		p.command(entityId, command, gsetRequest(&crdt.GSetAdd{Key: entityId, Value: values[0]}))
	})
}
