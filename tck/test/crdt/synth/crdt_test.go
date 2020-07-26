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
	"log"
	"net"
	"testing"
	"time"

	"github.com/cloudstateio/go-support/cloudstate"
	"github.com/cloudstateio/go-support/cloudstate/crdt"
	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	crdt2 "github.com/cloudstateio/go-support/tck/crdt"
	crdt3 "github.com/cloudstateio/go-support/tck/proto/crdt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const serviceName = "crdt.TckCrdt"

func TestCRDT(t *testing.T) {
	_, conn, teardown := setup(t)
	defer teardown()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("Entity Discovery should find the SyntheticCRDTs service", func(t *testing.T) {
		edc := protocol.NewEntityDiscoveryClient(conn)
		discover, err := edc.Discover(ctx, &protocol.ProxyInfo{
			ProtocolMajorVersion: 1,
			ProtocolMinorVersion: 0,
			ProxyName:            "p1",
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
		client, err := protocol.NewCrdtClient(conn).Handle(ctx)
		if err != nil {
			t.Fatal(err)
		}
		p := proxy{t: t, h: client}
		t.Run("send a CrdtInit should not fail", func(t *testing.T) {
			p.sendInit(&protocol.CrdtInit{
				ServiceName: serviceName,
				EntityId:    "gcounter-1",
			})
		})
		t.Run("incrementing a GCounter should emit a client action and create state action", func(t *testing.T) {
			out, err := p.sendRecvCmd(command{
				&protocol.Command{EntityId: "gcounter-1", Name: "IncrementGCounter"},
				&crdt3.GCounterIncrement{Key: "gcounter-1", Value: 7},
			})
			if err != nil {
				t.Fatal(err)
			}
			switch m := out.Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				// we expect a client action
				value := crdt3.GCounterValue{}
				err = encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
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
			out, err := p.sendRecvCmd(command{
				&protocol.Command{EntityId: "gcounter-1", Name: "IncrementGCounter"},
				&crdt3.GCounterIncrement{Key: "gcounter-1", Value: 7},
			})
			if err != nil {
				t.Fatal(err)
			}
			switch m := out.Message.(type) {
			case *protocol.CrdtStreamOut_Reply:
				value := crdt3.GCounterValue{}
				err = encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), &value)
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
		t.Run("GetGCounter should return the counters value", func(t *testing.T) {
			out, err := p.sendRecvCmd(
				command{
					&protocol.Command{EntityId: "gcounter-1", Name: "GetGCounter"},
					&crdt3.Get{Key: "gcounter-1"},
				},
			)
			if err != nil {
				t.Fatal(err)
			}
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
			out, err := p.sendRecvCmd(
				command{
					&protocol.Command{EntityId: "gcounter-1", Name: "GetGCounter"},
					&crdt3.Get{Key: "gcounter-1"},
				},
			)
			if err != nil {
				t.Fatal(err)
			}
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

			out, err := p.sendRecvCmd(
				command{
					&protocol.Command{EntityId: "gcounter-1", Name: "GetGCounter"},
					&crdt3.Get{Key: "gcounter-1"},
				},
			)
			if err != nil {
				t.Fatal(err)
			}
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
			out, err := p.sendRecvCmd(command{
				&protocol.Command{EntityId: "gcounter-2", Name: "GetGCounter"},
				&crdt3.Get{Key: "gcounter-2"},
			})
			if err != nil {
				t.Fatal(err)
			}
			switch m := out.Message.(type) {
			case *protocol.CrdtStreamOut_Failure:
			default:
				t.Fatalf("got unexpected message: %+v", m)
			}
		})
	})
}

func setup(t *testing.T) (server *cloudstate.CloudState, conn *grpc.ClientConn, teardown func()) {
	t.Helper()
	server, err := cloudstate.New(protocol.Config{
		ServiceName:    "io.cloudstate.tck.Crdt", // the service name the proxy gets to know about
		ServiceVersion: "0.2.0",
	})
	if err != nil {
		log.Fatalf("cloudstate.New failed: %v", err)
	}
	err = server.RegisterCRDT(
		&crdt.Entity{
			ServiceName: serviceName, // this is the package + service(name) from the gRPC proto file.
			EntityFunc: func(id crdt.EntityId) crdt.EntityHandler {
				return crdt2.NewEntity(id)
			},
		},
		protocol.DescriptorConfig{
			Service: "tck_crdt.proto", // this is needed to find the descriptors with got for the service to be proxied.
		},
	)
	if err != nil {
		log.Fatalf("cloudstate.New failed: %v", err)
	}
	lis := bufconn.Listen(1024 * 1024)
	go func() {
		if err := server.RunWithListener(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()

	// client connection
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	conn, err = grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	return server, conn, func() {
		conn.Close()
		server.Stop()
	}
}
