package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/cloudstateio/go-support/cloudstate"
	"github.com/cloudstateio/go-support/cloudstate/crdt"
	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/tck/crdts"
	"github.com/golang/protobuf/ptypes/any"
)

type CRDTs struct {
	count  *crdt.GCounter
	online *crdt.Vote
	crash  bool
}

func Set(ctx *crdt.Context, c crdt.CRDT) {
	ctx.Instance.(*CRDTs).count = c.(*crdt.GCounter)
}

func Default(c *crdt.Context) crdt.CRDT {
	if strings.HasPrefix(c.EntityId.String(), "gcounter-") {
		i := c.Instance.(*CRDTs)
		i.count = crdt.NewGCounter()
		return i.count
	}
	c.Fail(errors.New("unknown entity type"))
	return nil
}

func (p *CRDTs) StreamedCommand(c *crdt.CommandContext, name string, cmd interface{}) (*any.Any, error) {
	if p.crash {
		panic("forced crash")
	}
	fmt.Printf("got: %+v, %v\n", name, cmd)
	switch name {
	case "IncrementGCounter":
		switch u := cmd.(type) {
		case *crdts.UpdateCounter:
			defer func() { p.crash = true }()
			p.count.Increment(uint64(u.Value))
			return encoding.MarshalAny(&crdts.CounterValue{Value: int64(p.count.Value())})
		}
	case "GetGCounter":
		switch cmd.(type) {
		case *crdts.Get:
			return encoding.MarshalAny(&crdts.CounterValue{Value: int64(p.count.Value())})
		}
	}
	return nil, nil
}

func main() {
	server, err := cloudstate.New(cloudstate.Config{
		ServiceName:    "com.example.crdts.CrdtExample",
		ServiceVersion: "0.1.0",
	})
	if err != nil {
		log.Fatalf("cloudstate.New failed: %v", err)
	}
	err = server.RegisterCrdt(
		&crdt.Entity{
			ServiceName: "com.example.crdts.CrdtExample",
			EntityFunc:  func(id crdt.EntityId) interface{} { return &CRDTs{} },
			SetFunc:     Set,
			DefaultFunc: Default,
			CommandFunc: func(p interface{}, ctx *crdt.CommandContext, name string, msg interface{}) (*any.Any, error) {
				return p.(*CRDTs).StreamedCommand(ctx, name, msg)
			},
		},
		cloudstate.DescriptorConfig{
			Service: "crdts/crdt-example.proto",
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
