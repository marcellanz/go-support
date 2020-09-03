package synth

import (
	"context"
	"log"
	"net"
	"testing"

	"github.com/cloudstateio/go-support/cloudstate"
	"github.com/cloudstateio/go-support/cloudstate/crdt"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	crdt2 "github.com/cloudstateio/go-support/tck/crdt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const serviceName = "crdt.TckCrdt"

type server struct {
	t              *testing.T
	server         *cloudstate.CloudState
	conn           *grpc.ClientConn
	lis            *bufconn.Listener
	teardownServer func()
	teardownClient func()
	serviceName    string
}

func newServer(t *testing.T) *server {
	t.Helper()
	s := server{t: t}
	if s.t == nil {
		panic("not test context defined")
	}
	server, err := cloudstate.New(protocol.Config{
		ServiceName:    "io.cloudstate.tck.Crdt", // the service name the proxy gets to know about
		ServiceVersion: "0.2.0",
	})
	if err != nil {
		s.t.Fatal(err)
	}
	s.server = server
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
		s.t.Fatal(err)
	}
	s.lis = bufconn.Listen(1024 * 1024)
	s.teardownServer = func() {
		s.server.Stop()
	}
	go func() {
		if err := server.RunWithListener(s.lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
	s.newClientConn()
	return &s
}

func (s *server) newClientConn() {
	if s.conn != nil && s.teardownClient != nil {
		s.teardownClient()
	}
	// client
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return s.lis.Dial()
	}), grpc.WithInsecure())
	s.conn = conn
	if err != nil {
		s.t.Fatalf("Failed to dial bufnet: %v", err)
	}
	s.teardownClient = func() {
		s.conn.Close()
	}
}

func (s *server) teardown() {
	s.teardownClient()
	s.teardownServer()
}
