package synth

import (
	"context"
	"testing"
	"time"

	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/cloudstateio/go-support/tck/proto/crdt"
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
		t.Run("emit client action and create state action", func(t *testing.T) {
			tr := tester{t}
			switch m := p.command(
				entityId, command, orsetRequest(&crdt.ORSetAdd{Value: &crdt.AnySupportType{Value: &crdt.AnySupportType_AnyValue{AnyValue: encoding.Struct(pair{"one", 1})}}}),
			).Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
			default:
				tr.unexpected(m)
			}
		})
	})
}
