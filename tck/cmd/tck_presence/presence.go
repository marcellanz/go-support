package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/cloudstateio/go-support/cloudstate"
	"github.com/cloudstateio/go-support/cloudstate/crdt"
	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/cloudstateio/go-support/tck/presence"
	"github.com/golang/protobuf/ptypes/any"
)

type Presence struct {
}

// a command
func (p *Presence) Connect(c *crdt.CommandContext) *any.Any {
	if c.CRDT() != nil {
		if err := c.SetCRDT(crdt.NewVote()); err != nil {
			panic(err)
		}
	}
	flag := c.CRDT().(*crdt.Flag)

	// for a streamed command, this would be called once, the first time.
	// but the chance function solely for every command running, it might capture
	//
	c.ChangeFunc(func(ctx *crdt.CommandContext) (*any.Any, error) {
		flag.Enable()
		return nil, nil
	})

	c.CancelFunc(func(c *crdt.CommandContext) error {
		c.SideEffect(&protocol.SideEffect{
			ServiceName: "Service1",
			CommandName: "method1",
			Payload:     encoding.String("arg1"),
			Synchronous: true,
		})
		return nil
	})

	if true {
		c.Fail(fmt.Errorf("its a failure"))
	}
	if false {
		c.End()
	}
	return nil
}

func (p *Presence) StreamedCommand(c *crdt.CommandContext, name string, cmd interface{}) (*any.Any, error) {
	if !c.Streamed() {
		panic("I thought it is streamed")
	}
	switch u := cmd.(type) {
	case *presence.User:
		switch name {
		case "Connect":
			if u.GetName() == "jimmy" {
				c.Fail(errors.New("its jimmy"))
			}
		case "Monitor":
			u.GetName()
		}
	}
	return nil, nil
}

func main() {
	server, err := cloudstate.New(protocol.Config{
		ServiceName:    "presence",
		ServiceVersion: "0.1.0",
	})
	if err != nil {
		log.Fatalf("cloudstate.New failed: %v", err)
	}
	err = server.RegisterCRDT(
		&crdt.Entity{
			ServiceName: "presence",
			EntityFunc:  func(id crdt.EntityId) interface{} { return &Presence{} },
			DefaultFunc: func(c *crdt.Context) crdt.CRDT { return crdt.NewFlag() },
			CommandFunc: func(p interface{}, ctx *crdt.CommandContext, name string, msg interface{}) (*any.Any, error) {
				return p.(*Presence).StreamedCommand(ctx, name, msg)
			},
		},
		protocol.DescriptorConfig{
			Service: "presence.proto",
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
