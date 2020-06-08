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

func (cs *CRDTs) crashNextTime(id crdt.EntityId) {
	if cs.crashedFor[id] == false {
		cs.crashedFor[id] = true
	}
}

func Set(ctx *crdt.Context, c crdt.CRDT) {
	switch v := c.(type) {
	case *crdt.GCounter:
		ctx.Instance.(*CRDTs).gCounter = v
	case *crdt.PNCounter:
		ctx.Instance.(*CRDTs).pnCounter = v
	case *crdt.GSet:
		ctx.Instance.(*CRDTs).gSet = v
	case *crdt.ORSet:
		ctx.Instance.(*CRDTs).orSet = v
	case *crdt.Vote:
		ctx.Instance.(*CRDTs).vote = v
	}
}

func Default(c *crdt.Context) crdt.CRDT {
	if strings.HasPrefix(c.EntityId.String(), "gcounter-") {
		i := c.Instance.(*CRDTs)
		i.gCounter = crdt.NewGCounter()
		return i.gCounter
	}
	if strings.HasPrefix(c.EntityId.String(), "pncounter-") {
		i := c.Instance.(*CRDTs)
		i.pnCounter = crdt.NewPNCounter()
		return i.pnCounter
	}
	if strings.HasPrefix(c.EntityId.String(), "gset-") {
		i := c.Instance.(*CRDTs)
		i.gSet = crdt.NewGSet()
		return i.gSet
	}
	if strings.HasPrefix(c.EntityId.String(), "orset-") {
		i := c.Instance.(*CRDTs)
		i.orSet = crdt.NewORSet()
		return i.orSet
	}
	c.Fail(errors.New("unknown entity type"))
	return nil
}

func (cs *CRDTs) Command(ctx *crdt.CommandContext, name string, cmd interface{}) (*any.Any, error) {
	if cs.crashedFor[ctx.EntityId] {
		cs.crashedFor[ctx.EntityId] = true
		panic("forced crash")
	}
	fmt.Printf("got: %+v, %v\n", name, cmd)
	switch name {
	case "IncrementGCounter":
		switch c := cmd.(type) {
		case *crdts.UpdateCounter:
			defer cs.crashNextTime(ctx.EntityId)
			cs.gCounter.Increment(uint64(c.GetValue()))
			return encoding.MarshalAny(&crdts.CounterValue{Value: int64(cs.gCounter.Value())})
		}
	case "GetGCounter":
		switch cmd.(type) {
		case *crdts.Get:
			return encoding.MarshalAny(&crdts.CounterValue{Value: int64(cs.gCounter.Value())})
		}
	case "UpdatePNCounter":
		switch c := cmd.(type) {
		case *crdts.UpdateCounter:
			defer cs.crashNextTime(ctx.EntityId)
			cs.pnCounter.Increment(c.GetValue())
			return encoding.MarshalAny(&crdts.CounterValue{Value: cs.pnCounter.Value()})
		}
	case "GetPNCounter":
		switch cmd.(type) {
		case *crdts.Get:
			return encoding.MarshalAny(&crdts.CounterValue{Value: cs.pnCounter.Value()})
		}
	case "MutateGSet":
		switch m := cmd.(type) {
		case *crdts.MutateSet:
			defer cs.crashNextTime(ctx.EntityId)
			for _, v := range m.GetAdd() {
				x, err := encoding.MarshalAny(v)
				if err != nil {
					return nil, err
				}
				cs.gSet.Add(x)
			}
			if len(m.GetRemove()) > 0 {
				panic("a growing set can't remove items")
			}
			if m.Clear {
				panic("a growing set can't be cleared")
			}
			return encoding.MarshalAny(&crdts.SetSize{Size: int32(cs.gSet.Size())})
		}
	case "GetGSet":
		switch cmd.(type) {
		case *crdts.Get:
			sv := &crdts.SetValue{}
			for _, x := range cs.gSet.Value() {
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
			defer cs.crashNextTime(ctx.EntityId)
			for _, v := range m.GetAdd() {
				x, err := encoding.MarshalAny(v)
				if err != nil {
					return nil, err
				}
				cs.orSet.Add(x)
			}
			for _, v := range m.GetRemove() {
				x, err := encoding.MarshalAny(v)
				if err != nil {
					return nil, err
				}
				cs.orSet.Remove(x)
			}
			return encoding.MarshalAny(&crdts.SetSize{Size: int32(cs.orSet.Size())})
		}
	case "GetORSet":
		switch cmd.(type) {
		case *crdts.Get:
			sv := &crdts.SetValue{}
			for _, x := range cs.orSet.Value() {
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
	server, err := cloudstate.New(cloudstate.Config{
		ServiceName:    "com.example.crdts.CrdtExample0",
		ServiceVersion: "0.2.0",
	})
	if err != nil {
		log.Fatalf("cloudstate.New failed: %v", err)
	}
	err = server.RegisterCrdt(
		&crdt.Entity{
			ServiceName: "com.example.crdts.CrdtExample",
			EntityFunc:  func(id crdt.EntityId) interface{} { return newCRDTs() },
			SetFunc:     Set,
			DefaultFunc: Default,
			CommandFunc: func(entity interface{}, ctx *crdt.CommandContext, name string, msg interface{}) (*any.Any, error) {
				return entity.(*CRDTs).Command(ctx, name, msg)
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
