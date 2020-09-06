package synth

import (
	"context"
	"testing"
	"time"

	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/cloudstateio/go-support/tck/proto/crdt"
)

func TestCRDTORMap(t *testing.T) {
	s := newServer(t)
	defer s.teardown()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("ORMap", func(t *testing.T) {
		entityId := "ormap-1"
		command := "ProcessORMap"
		p := newProxy(ctx, s)
		defer p.teardown()

		p.init(&protocol.CrdtInit{ServiceName: serviceName, EntityId: entityId})
		t.Run("Get", func(t *testing.T) {
			tr := tester{t}
			switch m := p.command(entityId, command,
				ormapRequest(&crdt.Get{}),
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
			default:
				tr.unexpected(m)
			}
		})
		t.Run("Set – GCounter", func(t *testing.T) {
			tr := tester{t}
			switch m := p.command(entityId, command,
				ormapRequest(&crdt.ORMapSet{
					EntryKey: encoding.String("one"),
					Request: &crdt.ORMapSet_GCounterRequest{
						GCounterRequest: gcounterRequest(&crdt.GCounterIncrement{Value: 1}),
					},
				}),
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				tr.expectedNotNil(m.Reply.GetStateAction().GetCreate())
			default:
				tr.unexpected(m)
			}
			switch m := p.command(entityId, command,
				ormapRequest(&crdt.ORMapSet{
					EntryKey: encoding.String("one"),
					Request: &crdt.ORMapSet_GCounterRequest{
						GCounterRequest: gcounterRequest(&crdt.GCounterIncrement{Value: 1}),
					},
				}),
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				tr.expectedNotNil(m.Reply.GetStateAction().GetUpdate())
			default:
				tr.unexpected(m)
			}
		})
	})
}
