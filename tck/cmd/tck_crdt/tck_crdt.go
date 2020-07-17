package main

import (
	"log"

	"github.com/cloudstateio/go-support/cloudstate"
	"github.com/cloudstateio/go-support/cloudstate/crdt"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	tck "github.com/cloudstateio/go-support/tck/crdt"
)

func main() {
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
				return tck.NewEntity(id)
			},
		},
		protocol.DescriptorConfig{
			Service: "tck_crdt.proto", // this is needed to find the descriptors with got for the service to be proxied.
		},
	)
	if err != nil {
		log.Fatalf("Cloudstate failed to register entity: %v", err)
	}
	err = server.Run()
	if err != nil {
		log.Fatalf("Cloudstate failed to run: %v", err)
	}
}
