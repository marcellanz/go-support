package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/cloudstateio/go-support/cloudstate"
	"github.com/cloudstateio/go-support/cloudstate/crdt"
	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/cloudstateio/go-support/tck/crdts"
	"github.com/golang/protobuf/proto"

	"github.com/golang/protobuf/ptypes/any"
)

type CRDTs struct {
	gCounter   *crdt.GCounter
	vote       *crdt.Vote
	pnCounter  *crdt.PNCounter
	gSet       *crdt.GSet
	orSet      *crdt.ORSet
	crashedFor map[crdt.EntityId]bool
}

func newCRDTs() *CRDTs {
	return &CRDTs{crashedFor: make(map[crdt.EntityId]bool)}
}

func (entity *CRDTs) crashNextTime(id crdt.EntityId) {
	if entity.crashedFor[id] == false {
		entity.crashedFor[id] = true
	}
}

func (entity *CRDTs) Set(_ *crdt.Context, c crdt.CRDT) {
	switch v := c.(type) {
	case *crdt.GCounter:
		entity.gCounter = v
	case *crdt.PNCounter:
		entity.pnCounter = v
	case *crdt.GSet:
		entity.gSet = v
	case *crdt.ORSet:
		entity.orSet = v
	case *crdt.Vote:
		entity.vote = v
	}
}

func (entity *CRDTs) Default(ctx *crdt.Context) crdt.CRDT {
	if strings.HasPrefix(ctx.EntityId.String(), "gcounter-") {
		entity.gCounter = crdt.NewGCounter()
		return entity.gCounter
	}
	if strings.HasPrefix(ctx.EntityId.String(), "pncounter-") {
		entity.pnCounter = crdt.NewPNCounter()
		return entity.pnCounter
	}
	if strings.HasPrefix(ctx.EntityId.String(), "gset-") {
		entity.gSet = crdt.NewGSet()
		return entity.gSet
	}
	if strings.HasPrefix(ctx.EntityId.String(), "orset-") {
		entity.orSet = crdt.NewORSet()
		return entity.orSet
	}
	ctx.Fail(errors.New("unknown entity type"))
	return nil
}

func (entity *CRDTs) HandleCommand(ctx *crdt.CommandContext, name string, cmd proto.Message) (*any.Any, error) {
	if entity.crashedFor[ctx.EntityId] {
		entity.crashedFor[ctx.EntityId] = true
		panic("forced crash")
	}
	fmt.Printf("got: %+v, %v\n", name, cmd)
	switch name {
	case "IncrementGCounter":
		switch c := cmd.(type) {
		case *crdts.UpdateCounter:
			defer entity.crashNextTime(ctx.EntityId)
			entity.gCounter.Increment(uint64(c.GetValue()))
			return encoding.MarshalAny(&crdts.CounterValue{Value: int64(entity.gCounter.Value())})
		}
	case "GetGCounter":
		switch cmd.(type) {
		case *crdts.Get:
			return encoding.MarshalAny(&crdts.CounterValue{Value: int64(entity.gCounter.Value())})
		}
	case "UpdatePNCounter":
		switch c := cmd.(type) {
		case *crdts.UpdateCounter:
			defer entity.crashNextTime(ctx.EntityId)
			entity.pnCounter.Increment(c.GetValue())
			return encoding.MarshalAny(&crdts.CounterValue{Value: entity.pnCounter.Value()})
		}
	case "GetPNCounter":
		switch cmd.(type) {
		case *crdts.Get:
			return encoding.MarshalAny(&crdts.CounterValue{Value: entity.pnCounter.Value()})
		}
	case "MutateGSet":
		switch m := cmd.(type) {
		case *crdts.MutateSet:
			defer entity.crashNextTime(ctx.EntityId)
			for _, v := range m.GetAdd() {
				x, err := encoding.MarshalAny(v)
				if err != nil {
					return nil, err
				}
				entity.gSet.Add(x)
			}
			if len(m.GetRemove()) > 0 {
				panic("a growing set can't remove items")
			}
			if m.Clear {
				panic("a growing set can't be cleared")
			}
			return encoding.MarshalAny(&crdts.SetSize{Size: int32(entity.gSet.Size())})
		}
	case "GetGSet":
		switch cmd.(type) {
		case *crdts.Get:
			sv := &crdts.SetValue{}
			for _, x := range entity.gSet.Value() {
				v := &crdts.SomeValue{}
				if err := encoding.UnmarshalAny(x, v); err != nil {
					return nil, err
				}
				sv.Items = append(sv.Items, v)
			}
			return encoding.MarshalAny(sv)
		}
	case "MutateORSet":
		switch m := cmd.(type) {
		case *crdts.MutateSet:
			defer entity.crashNextTime(ctx.EntityId)
			for _, v := range m.GetAdd() {
				x, err := encoding.MarshalAny(v)
				if err != nil {
					return nil, err
				}
				entity.orSet.Add(x)
			}
			for _, v := range m.GetRemove() {
				x, err := encoding.MarshalAny(v)
				if err != nil {
					return nil, err
				}
				entity.orSet.Remove(x)
			}
			return encoding.MarshalAny(&crdts.SetSize{Size: int32(entity.orSet.Size())})
		}
	case "GetORSet":
		switch cmd.(type) {
		case *crdts.Get:
			sv := &crdts.SetValue{}
			for _, x := range entity.orSet.Value() {
				v := &crdts.SomeValue{}
				if err := encoding.UnmarshalAny(x, v); err != nil {
					return nil, err
				}
				sv.Items = append(sv.Items, v)
			}
			return encoding.MarshalAny(sv)
		}
	}

	return nil, nil
}

func main() {
	server, err := cloudstate.New(protocol.Config{
		ServiceName:    "com.example.crdts.CrdtExample0",
		ServiceVersion: "0.2.0",
	})
	if err != nil {
		log.Fatalf("cloudstate.New failed: %v", err)
	}
	err = server.RegisterCRDT(
		&crdt.Entity{
			ServiceName: "com.example.crdts.CrdtExample",
			EntityFunc:  func(id crdt.EntityId) crdt.EntityHandler { return newCRDTs() },
		},
		protocol.DescriptorConfig{
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
