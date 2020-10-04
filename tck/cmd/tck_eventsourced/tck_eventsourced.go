package main

import (
	"log"

	"github.com/cloudstateio/go-support/cloudstate"
	"github.com/cloudstateio/go-support/cloudstate/eventsourced"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	eventsourced2 "github.com/cloudstateio/go-support/tck/eventsourced"
	"github.com/cloudstateio/go-support/tck/shoppingcart"
)

func main() {
	server, err := cloudstate.New(protocol.Config{
		ServiceName:    "cloudstate.tck.model.EventSourcedTckModel", // the servicename the proxy gets to know about
		ServiceVersion: "0.2.0",
	})
	if err != nil {
		log.Fatalf("cloudstate.New failed: %v", err)
	}
	err = server.RegisterEventSourced(
		&eventsourced.Entity{
			ServiceName:   "cloudstate.tck.model.EventSourcedTckModel",
			PersistenceID: "event-sourced-tck-model",
			SnapshotEvery: 5,
			EntityFunc:    eventsourced2.NewTestModel,
		}, protocol.DescriptorConfig{
			Service: "eventsourced.proto",
		},
	)
	if err != nil {
		log.Fatalf("CloudState failed to register entity: %v", err)
	}
	err = server.RegisterEventSourced(
		&eventsourced.Entity{
			ServiceName:   "cloudstate.tck.model.EventSourcedTwo",
			PersistenceID: "EventSourcedTwo",
			SnapshotEvery: 5,
			EntityFunc:    eventsourced2.NewTestModelTwo,
		}, protocol.DescriptorConfig{
			Service: "eventsourced.proto",
		},
	)
	if err != nil {
		log.Fatalf("CloudState failed to register entity: %v", err)
	}
	err = server.RegisterEventSourced(&eventsourced.Entity{
		ServiceName:   "com.example.shoppingcart.ShoppingCart",
		PersistenceID: "ShoppingCart",
		EntityFunc:    shoppingcart.NewShoppingCart,
	}, protocol.DescriptorConfig{
		Service: "shoppingcart/shoppingcart.proto",
	}.AddDomainDescriptor("domain.proto"))
	if err != nil {
		log.Fatalf("CloudState failed to register entity: %v", err)
	}
	err = server.Run()
	if err != nil {
		log.Fatalf("CloudState failed to run: %v", err)
	}

}
