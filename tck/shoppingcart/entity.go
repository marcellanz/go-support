package shoppingcart

import (
	"errors"
	"fmt"

	"github.com/cloudstateio/go-support/cloudstate/eventsourced"
	domain "github.com/cloudstateio/go-support/tck/shoppingcart/persistence"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
)

// A Cloudstate event sourced entity.
type ShoppingCart struct {
	// our domain object
	cart []*domain.LineItem
}

// NewShoppingCart returns a new and initialized instance of the ShoppingCart entity.
func NewShoppingCart(eventsourced.EntityId) eventsourced.EntityHandler {
	return &ShoppingCart{
		cart: make([]*domain.LineItem, 0),
	}
}

// ItemAdded is a event handler function for the ItemAdded event.
func (sc *ShoppingCart) ItemAdded(added *domain.ItemAdded) error { // TODO: enable handling for values
	if added.Item.GetName() == "FAIL" {
		return errors.New("boom: forced an unexpected error")
	}
	if item, _ := sc.find(added.Item.ProductId); item != nil {
		item.Quantity += added.Item.Quantity
		return nil
	}
	sc.cart = append(sc.cart, &domain.LineItem{
		ProductId: added.Item.ProductId,
		Name:      added.Item.Name,
		Quantity:  added.Item.Quantity,
	})
	return nil
}

// ItemRemoved is a event handler function for the ItemRemoved event.
func (sc *ShoppingCart) ItemRemoved(removed *domain.ItemRemoved) error {
	if !sc.remove(removed.ProductId) {
		return errors.New("unable to remove product")
	}
	return nil
}

// HandleEvent lets us handle events by ourselves.
func (sc *ShoppingCart) HandleEvent(_ *eventsourced.Context, event interface{}) error {
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
func (sc *ShoppingCart) AddItem(ctx *eventsourced.Context, li *AddLineItem) (*empty.Empty, error) {
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
func (sc *ShoppingCart) RemoveItem(ctx *eventsourced.Context, li *RemoveLineItem) (*empty.Empty, error) {
	if item, _ := sc.find(li.GetProductId()); item == nil {
		return nil, fmt.Errorf("cannot remove item %s because it is not in the cart", li.GetProductId())
	}
	ctx.Emit(&domain.ItemRemoved{ProductId: li.ProductId})
	return &empty.Empty{}, nil
}

// GetCart implements the GetCart command handling of the shopping cart service.
func (sc *ShoppingCart) GetCart(*eventsourced.Context, *GetShoppingCart) (*Cart, error) {
	cart := &Cart{}
	for _, item := range sc.cart {
		cart.Items = append(cart.Items, &LineItem{
			ProductId: item.ProductId,
			Name:      item.Name,
			Quantity:  item.Quantity,
		})
	}
	return cart, nil
}

// HandleCommand is the entities command handler implemented by the shopping cart.
func (sc *ShoppingCart) HandleCommand(ctx *eventsourced.Context, name string, cmd proto.Message) (proto.Message, error) {
	switch c := cmd.(type) {
	case *GetShoppingCart:
		return sc.GetCart(ctx, c)
	case *RemoveLineItem:
		return sc.RemoveItem(ctx, c)
	case *AddLineItem:
		return sc.AddItem(ctx, c)
	default:
		return nil, nil
	}
}

// Snapshot returns the current state of the shopping cart.
func (sc *ShoppingCart) Snapshot(*eventsourced.Context) (snapshot interface{}, err error) {
	return &domain.Cart{
		Items: append(make([]*domain.LineItem, 0, len(sc.cart)), sc.cart...),
	}, nil
}

// HandleSnapshot applies given snapshot to be the current state.
func (sc *ShoppingCart) HandleSnapshot(ctx *eventsourced.Context, snapshot interface{}) error {
	switch value := snapshot.(type) {
	case *domain.Cart: // missed because of non-pointer type!!!
		sc.cart = append(sc.cart[:0], value.Items...)
		return nil
	default:
		return fmt.Errorf("unknown snapshot type: %v", value)
	}
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
// An ok flag is returned to indicate that the product was present and removed.
func (sc *ShoppingCart) remove(productId string) (ok bool) {
	if item, i := sc.find(productId); item != nil {
		// remove and re-slice
		copy(sc.cart[i:], sc.cart[i+1:])
		sc.cart = sc.cart[:len(sc.cart)-1]
		return true
	}
	return false
}
