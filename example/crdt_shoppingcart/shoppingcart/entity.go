package shoppingcart

import (
	"errors"

	"github.com/cloudstateio/go-support/cloudstate/crdt"
	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
)

//go:generate protoc --go_out=plugins=grpc,paths=source_relative:./example/crdt_shoppingcart/shoppingcart --proto_path=protobuf/protocol --proto_path=protobuf/frontend --proto_path=protobuf/frontend/cloudstate --proto_path=protobuf/proxy --proto_path=example/crdt_shoppingcart/shoppingcart proto

type ShoppingCart struct {
	// items *crdt.LWWRegister
	items *crdt.ORMap
}

func (s *ShoppingCart) getCart() (*Cart, error) {
	items := &Cart{}
	for _, state := range s.items.Values() {
		var item LineItem
		if err := encoding.DecodeStruct(state.GetLwwregister().GetValue(), &item); err != nil {
			return nil, err
		}
		items.Items = append(items.Items, &item)
	}
	return items, nil
}

func (s *ShoppingCart) HandleCommand(ctx *crdt.CommandContext, name string, msg proto.Message) (*any.Any, error) {
	switch name {
	case "WatchCart":
		ctx.ChangeFunc(func(c *crdt.CommandContext) (*any.Any, error) {
			cart, err := s.getCart()
			if err != nil {
				return nil, err
			}
			return encoding.MarshalAny(cart)
		})
		cart, err := s.getCart()
		if err != nil {
			return nil, err
		}
		return encoding.MarshalAny(cart)
	}

	switch m := msg.(type) {
	case *GetShoppingCart:
		cart, err := s.getCart()
		if err != nil {
			return nil, err
		}
		return encoding.MarshalAny(cart)
	case *AddLineItem:
		if m.GetQuantity() <= 0 {
			return nil, errors.New("cannot add a negative quantity of items")
		}

		key := encoding.String(m.GetProductId())
		reg, err := s.items.LWWRegister(key)
		if err != nil {
			return nil, err
		}
		item, err := encoding.MarshalAny(&LineItem{
			ProductId: m.GetProductId(),
			Name:      m.GetName(),
			Quantity:  m.GetQuantity(),
		})
		if err != nil {
			return nil, err
		}
		if reg == nil {
			reg = crdt.NewLWWRegister(item)
		} else {
			reg.Set(item)
		}
		s.items.Set(key, reg)
		return encoding.Empty, nil
	default:
		return nil, nil
	}
}

func (s *ShoppingCart) Default(ctx *crdt.Context) (crdt.CRDT, error) {
	return crdt.NewLWWRegister(nil), nil
}

func (s *ShoppingCart) Set(ctx *crdt.Context, state crdt.CRDT) error {
	switch c := state.(type) {
	case *crdt.ORMap:
		s.items = c
		return nil
	default:
		return errors.New("unable to set state")
	}
}
