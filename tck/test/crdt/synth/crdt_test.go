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

	"github.com/cloudstateio/go-support/cloudstate"
	"github.com/cloudstateio/go-support/cloudstate/crdt"
	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	crdt2 "github.com/cloudstateio/go-support/tck/crdt"
	crdt3 "github.com/cloudstateio/go-support/tck/proto/crdt"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func TestCRDT(t *testing.T) {
	server, err := cloudstate.New(protocol.Config{
		ServiceName:    "io.cloudstate.tck.Crdt", // the servicename the proxy gets to know about
		ServiceVersion: "0.2.0",
	})
	if err != nil {
		log.Fatalf("cloudstate.New failed: %v", err)
	}
	err = server.RegisterCRDT(
		&crdt.Entity{
			ServiceName: "crdt.TckCrdt", // this is the package + service(name) from the gRPC proto file.
			EntityFunc: func(id crdt.EntityId) crdt.EntityHandler {
				return crdt2.NewEntity(id)
			},
		},
		protocol.DescriptorConfig{
			Service: "tck_crdt.proto", // this is needed to find the descriptors with got for the service to be proxied.
		},
	)
	lis := bufconn.Listen(1024 * 1024)
	defer server.Stop()
	go func() {
		if err := server.RunWithListener(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()

	// client
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	edc := protocol.NewEntityDiscoveryClient(conn)
	discover, err := edc.Discover(ctx, &protocol.ProxyInfo{
		ProtocolMajorVersion: 0,
		ProtocolMinorVersion: 0,
		ProxyName:            "p1",
		ProxyVersion:         "0.0.0",
		SupportedEntityTypes: []string{protocol.EventSourced, protocol.CRDT},
	})
	if err != nil {
		t.Fatal(err)
	}

	// discovery
	if l := len(discover.GetEntities()); l != 1 {
		t.Fatalf("discover.Entities is:%d, should be: 1", l)
	}
	sn := discover.GetEntities()[0].GetServiceName()
	t.Run("Entity Discovery should find the SyntheticCRDTs service", func(t *testing.T) {
		if got, want := sn, "crdt.TckCrdt"; got != want {
			t.Fatalf("discover.Entities[0].ServiceName = %s; want: %s", got, want)
		}
	})

	t.Run("GCounter", func(t *testing.T) {
		cl := protocol.NewCrdtClient(conn)
		handler, err := cl.Handle(ctx)
		if err != nil {
			t.Fatal(err)
		}
		p := proxy{t: t, h: handler}

		t.Run("send a CrdtInit", func(t *testing.T) {
			p.sendInit(&protocol.CrdtInit{
				ServiceName: sn,
				EntityId:    "gcounter-1",
			})
		})
		t.Run("incrementing a gcounter should emit a client and create state action", func(t *testing.T) {
			out, err := p.sendRecvCmd(command{
				&protocol.Command{EntityId: "gcounter-1", Name: "IncrementGCounter"},
				&crdt3.GCounterIncrement{Key: "gcounter-1", Value: 7},
			},
			)
			if err != nil {
				t.Fatal(err)
			}

			// we expect a client action
			if f := out.GetFailure(); f != nil {
				t.Fatalf("got unexpected failure: %+v", f)
			}
			value := crdt3.GCounterValue{}
			err = encoding.UnmarshalAny(out.GetReply().GetClientAction().GetReply().GetPayload(), &value)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := value.GetValue(), uint64(7); got != want {
				t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
			}
			if got, want := out.GetReply().GetStateAction().GetCreate().GetGcounter().GetValue(), uint64(7); got != want {
				t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, out.GetReply())
			}
		})
		t.Run("a second increment should emit a client action and an update state action", func(t *testing.T) {
			out, err := p.sendRecvCmd(
				command{
					&protocol.Command{EntityId: "gcounter-1", Name: "IncrementGCounter"},
					&crdt3.GCounterIncrement{Key: "counter-1", Value: 7},
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			if f := out.GetFailure(); f != nil {
				t.Fatalf("got unexpected failure: %+v", f)
			}
			value := crdt3.GCounterValue{}
			err = encoding.UnmarshalAny(out.GetReply().GetClientAction().GetReply().GetPayload(), &value)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := value.GetValue(), uint64(14); got != want {
				t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, value)
			}
			if got, want := out.GetReply().GetStateAction().GetUpdate().GetGcounter().GetIncrement(), uint64(7); got != want {
				t.Fatalf("got = %v; wanted: %d, for value:%+v", got, want, out.GetReply())
			}
		})
	})
}

func compareClientAction(out *protocol.CrdtStreamOut, other *protocol.ClientAction) bool {
	switch m := out.Message.(type) {
	case *protocol.CrdtStreamOut_Reply:
		g := m.Reply.GetClientAction().GetFailure()
		o := other.GetFailure()
		if g == nil || o == nil {
			return false
		}
		if g.CommandId == 0 || o.CommandId == 0 {
			return false
		}
		if g.CommandId != o.CommandId {
			return false
		}
		if g.Description == "" || o.Description == "" {
			return false
		}
		if g.Description != o.Description {
			return false
		}
		return true
	case *protocol.CrdtStreamOut_StreamedMessage:
		//action := m.StreamedMessage.ClientAction
	}
	return false
}

type command struct {
	c *protocol.Command
	m proto.Message
}

type proxy struct {
	h               protocol.Crdt_HandleClient
	t               *testing.T
	commandSequence int64
}

func (p *proxy) sendInit(i *protocol.CrdtInit) {
	err := p.h.Send(&protocol.CrdtStreamIn{Message: &protocol.CrdtStreamIn_Init{
		Init: i,
	}})
	if err != nil {
		p.t.Fatal(err)
	}
}

func (p *proxy) sendRecvCmd(cmd command) (*protocol.CrdtStreamOut, error) {
	if cmd.c.Id == 0 {
		cmd.c.Id = p.commandSequence
		defer func() { p.commandSequence++ }()
	}
	any, err := encoding.MarshalAny(cmd.m)
	if err != nil {
		return nil, err
	}
	cmd.c.Payload = any
	err = p.h.Send(commandMsg(cmd.c))
	if err != nil {
		return nil, err
	}
	recv, err := p.h.Recv()
	if err != nil {
		return nil, err
	}
	checkCommandId(p.commandSequence, recv, p.t)
	return recv, err
}

func commandMsg(c *protocol.Command) *protocol.CrdtStreamIn {
	return &protocol.CrdtStreamIn{
		Message: &protocol.CrdtStreamIn_Command{
			Command: c,
		},
	}
}

func checkCommandId(cmdId int64, msg interface{}, t *testing.T) {
	switch m := msg.(type) {
	case *protocol.CrdtStreamOut:
		switch out := m.Message.(type) {
		case *protocol.CrdtStreamOut_Reply:
			if got, want := out.Reply.CommandId, cmdId; got != want {
				t.Fatalf("command = %v; wanted: %d, for message:%+v", got, want, out)
			}
		case *protocol.CrdtStreamOut_Failure:
			if got, want := out.Failure.CommandId, cmdId; got != want {
				t.Fatalf("command = %v; wanted: %d, for message:%+v", got, want, out)
			}
		case *protocol.CrdtStreamOut_StreamedMessage:
			if got, want := out.StreamedMessage.CommandId, cmdId; got != want {
				t.Fatalf("command = %v; wanted: %d, for message:%+v", got, want, out)
			}
		case *protocol.CrdtStreamOut_StreamCancelledResponse:
			if got, want := out.StreamCancelledResponse.CommandId, cmdId; got != want {
				t.Fatalf("command = %v; wanted: %d, for message:%+v", got, want, out)
			}
		default:
			t.Fatalf("unexpected message: %+v", m)
		}
	default:
		t.Fatalf("unexpected message: %+v", m)
	}
}
