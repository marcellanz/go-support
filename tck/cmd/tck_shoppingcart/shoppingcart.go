//
// Copyright 2019 Lightbend Inc.
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

// Package main implements an event sourced entity shopping cart example
package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/cloudstateio/go-support/cloudstate"
	"github.com/cloudstateio/go-support/cloudstate/eventsourced"
	"github.com/cloudstateio/go-support/tck/shoppingcart"
	domain "github.com/cloudstateio/go-support/tck/shoppingcart/persistence"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
)

// main creates a CloudState instance and registers the ShoppingCart
// as a event sourced entity.
func main() {
	server, err := cloudstate.New(cloudstate.Config{
		ServiceName:    "shopping-cart",
		ServiceVersion: "0.1.0",
	})
	if err != nil {
		log.Fatalf("CloudState.New failed: %v", err)
	}
	err = server.RegisterEventSourcedEntity(&eventsourced.EventSourcedEntity{
		ServiceName:   "com.example.shoppingcart.ShoppingCart",
		PersistenceID: "ShoppingCart",
		EntityFunc:    NewShoppingCart,
		CommandFunc: func(entity interface{}, ctx *eventsourced.Context, name string, msg proto.Message) (proto.Message, error) {
			return entity.(*ShoppingCart).HandleCommand(ctx, name, msg)
		},
		SnapshotFunc: func(entity interface{}) (snapshot interface{}, err error) {
			return entity.(*ShoppingCart).Snapshot()
		},
		SnapshotHandlerFunc: func(entity interface{}, ctx *eventsourced.Context, snapshot interface{}) error {
			return entity.(*ShoppingCart).HandleSnapshot(snapshot)
		},
		EventFunc: func(entity interface{}, ctx *eventsourced.Context, event interface{}) error {
			return entity.(*ShoppingCart).HandleEvent(ctx, event)
		},
	},
		cloudstate.DescriptorConfig{
			Service: "shoppingcart/shoppingcart.proto",
		}.AddDomainDescriptor("domain.proto"),
	)

	if err != nil {
		log.Fatalf("CloudState failed to register entity: %v", err)
	}
	err = server.Run()
	if err != nil {
		log.Fatalf("CloudState failed to run: %v", err)
	}
}

// A CloudState event sourced entity.
type ShoppingCart struct {
	// our domain object
	cart []*domain.LineItem
}

// NewShoppingCart returns a new and initialized instance of the ShoppingCart entity.
func NewShoppingCart(eventsourced.EntityId) interface{} {
	return &ShoppingCart{
		cart: make([]*domain.LineItem, 0),
	}
}

// ItemAdded is a event handler function for the ItemAdded event.
func (sc *ShoppingCart) ItemAdded(added *domain.ItemAdded) error { // TODO: enable handling for values
	if item, _ := sc.find(added.Item.ProductId); item != nil {
		item.Quantity += added.Item.Quantity
	} else {
		sc.cart = append(sc.cart, &domain.LineItem{
			ProductId: added.Item.ProductId,
			Name:      added.Item.Name,
			Quantity:  added.Item.Quantity,
		})
	}
	return nil
}

// ItemRemoved is a event handler function for the ItemRemoved event.
func (sc *ShoppingCart) ItemRemoved(removed *domain.ItemRemoved) error {
	if !sc.remove(removed.ProductId) {
		return errors.New("unable to remove product") // this should never happen
	}
	return nil
}

// Handle lets us handle events by ourselves.
//
// returns handle set to true if we have handled the event
// and any error that happened during the handling
func (sc *ShoppingCart) HandleEvent(ctx *eventsourced.Context, event interface{}) error {
	switch e := event.(type) {
	case *domain.ItemAdded:
		return sc.ItemAdded(e)
	case *domain.ItemRemoved:
		return sc.ItemRemoved(e)
	default:
		return nil
	}
}

// AddItem implements the AddItem command handling of the shopping cart service.
func (sc *ShoppingCart) AddItem(ctx *eventsourced.Context, li *shoppingcart.AddLineItem) (*empty.Empty, error) {
	if li.GetQuantity() <= 0 {
		return nil, fmt.Errorf("cannot add negative quantity of to item %s", li.GetProductId())
	}
	ctx.Emit(&domain.ItemAdded{
		Item: &domain.LineItem{
			ProductId: li.ProductId,
			Name:      li.Name,
			Quantity:  li.Quantity,
		}})
	return &empty.Empty{}, nil
}

// RemoveItem implements the RemoveItem command handling of the shopping cart service.
func (sc *ShoppingCart) RemoveItem(ctx *eventsourced.Context, li *shoppingcart.RemoveLineItem) (*empty.Empty, error) {
	if item, _ := sc.find(li.GetProductId()); item == nil {
		return nil, fmt.Errorf("cannot remove item %s because it is not in the cart", li.GetProductId())
	}
	ctx.Emit(&domain.ItemRemoved{ProductId: li.ProductId})
	return &empty.Empty{}, nil
}

// GetCart implements the GetCart command handling of the shopping cart service.
func (sc *ShoppingCart) GetCart(*eventsourced.Context, *shoppingcart.GetShoppingCart) (*shoppingcart.Cart, error) {
	cart := &shoppingcart.Cart{}
	for _, item := range sc.cart {
		cart.Items = append(cart.Items, &shoppingcart.LineItem{
			ProductId: item.ProductId,
			Name:      item.Name,
			Quantity:  item.Quantity,
		})
	}
	return cart, nil
}

func (sc *ShoppingCart) HandleCommand(ctx *eventsourced.Context, name string, cmd proto.Message) (proto.Message, error) {
	switch cmd := cmd.(type) {
	case *shoppingcart.GetShoppingCart:
		return sc.GetCart(ctx, cmd)
	case *shoppingcart.RemoveLineItem:
		return sc.RemoveItem(ctx, cmd)
	case *shoppingcart.AddLineItem:
		return sc.AddItem(ctx, cmd)
	default:
		return nil, nil
	}
}

func (sc *ShoppingCart) Snapshot() (snapshot interface{}, err error) {
	return domain.Cart{
		Items: append(make([]*domain.LineItem, 0, len(sc.cart)), sc.cart...),
	}, nil
}

func (sc *ShoppingCart) HandleSnapshot(snapshot interface{}) error {
	switch value := snapshot.(type) {
	case domain.Cart:
		sc.cart = append(sc.cart[:0], value.Items...)
	}
	return nil
}

// find finds a product in the shopping cart by productId and returns it as a LineItem.
func (sc *ShoppingCart) find(productId string) (item *domain.LineItem, index int) {
	for i, item := range sc.cart {
		if productId == item.ProductId {
			return item, i
		}
	}
	return nil, 0
}

// remove removes a product from the shopping cart.
//
// A ok flag is returned to indicate that the product was present and removed.
func (sc *ShoppingCart) remove(productId string) (ok bool) {
	if item, i := sc.find(productId); item != nil {
		// remove and re-slice
		copy(sc.cart[i:], sc.cart[i+1:])
		sc.cart = sc.cart[:len(sc.cart)-1]
		return true
	} else {
		return false
	}
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}
