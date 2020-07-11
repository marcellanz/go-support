package test

import (
	"context"
	"log"
	"net"
	"testing"

	"github.com/cloudstateio/go-support/cloudstate"
	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/eventsourced"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/cloudstateio/go-support/tck/shoppingcart"
	domain "github.com/cloudstateio/go-support/tck/shoppingcart/persistence"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func TestShoppingCart(t *testing.T) {
	server, _ := cloudstate.New(protocol.Config{
		ServiceName:    "shopping-cart",
		ServiceVersion: "9.9.8",
	})
	err := server.RegisterEventSourced(&eventsourced.Entity{
		ServiceName:   "com.example.shoppingcart.ShoppingCart",
		PersistenceID: "ShoppingCart",
		EntityFunc:    shoppingcart.NewShoppingCart,
		SnapshotEvery: 1,
	}, protocol.DescriptorConfig{
		Service: "shoppingcart/shoppingcart.proto",
	}.AddDomainDescriptor("domain.proto"))

	lis := bufconn.Listen(bufSize)
	defer server.Stop()
	go func() {
		if err := server.RunWithListener(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()

	// client
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	edc := protocol.NewEntityDiscoveryClient(conn)
	discover, err := edc.Discover(ctx, &protocol.ProxyInfo{
		ProtocolMajorVersion: 0,
		ProtocolMinorVersion: 0,
		ProxyName:            "p1",
		ProxyVersion:         "0.0.0",
		SupportedEntityTypes: []string{protocol.EventSourced, protocol.CRDT},
	})
	if err != nil {
		t.Fatal(err)
	}

	discover.GetServiceInfo().GetServiceName()
	// discovery
	if l := len(discover.GetEntities()); l != 1 {
		t.Fatalf("discover.Entities is:%d, should be: 1", l)
	}
	sn := discover.GetEntities()[0].GetServiceName()
	t.Run("Entity Discovery should find the shopping cart service", func(t *testing.T) {
		if sn != "com.example.shoppingcart.ShoppingCart" {
			t.Fatalf("discover.Entities[0].ServiceName is:%v, should be: %s", sn, "com.example.shoppingcart.ShoppingCart")
		}
	})

	cmdId := int64(0)
	t.Run("GetShoppingCart should fail without an init message", func(t *testing.T) {
		esc := protocol.NewEventSourcedClient(conn)
		handle, err := esc.Handle(ctx)
		if err != nil {
			t.Fatal(err)
		}
		r, err := sendRecvCmd(
			&protocol.Command{EntityId: "e1", Id: cmdId, Name: "GetShoppingCart"},
			&shoppingcart.GetShoppingCart{UserId: "user1"},
			handle,
		)
		if err != nil {
			t.Fatal(err)
		}
		switch m := r.Message.(type) {
		case *protocol.EventSourcedStreamOut_Failure:
			checkCommandId(cmdId, m, t)
		case *protocol.EventSourcedStreamOut_Reply:
			t.Fatal("a message should not be allowed to be received without a init message sent before")
		default:
			t.Fatalf("unexpected message: %+v", m)
		}
	})

	cmdId++
	t.Run("GetShoppingCart with init", func(t *testing.T) {
		esc := protocol.NewEventSourcedClient(conn)
		handle, err := esc.Handle(ctx)
		if err != nil {
			t.Fatal(err)
		}
		err = handle.Send(initMsg(&protocol.EventSourcedInit{
			ServiceName: sn,
			EntityId:    "e2",
		}))
		r, err := sendRecvCmd(
			&protocol.Command{EntityId: "e2", Id: cmdId, Name: "GetShoppingCart"},
			&shoppingcart.GetShoppingCart{UserId: "user2"},
			handle,
		)
		if err != nil {
			t.Fatal(err)
		}
		switch m := r.Message.(type) {
		case *protocol.EventSourcedStreamOut_Reply:
			checkCommandId(cmdId, m, t)
		case *protocol.EventSourcedStreamOut_Failure:
			t.Fatalf("expected reply but got: %+v", m.Failure)
		default:
			t.Fatalf("unexpected message: %+v", m)
		}
	})

	cmdId++
	t.Run("AddLineItem should return the same line item with GetCart", func(t *testing.T) {
		esc := protocol.NewEventSourcedClient(conn)
		handle, err := esc.Handle(ctx)
		if err != nil {
			t.Fatal(err)
		}
		err = handle.Send(initMsg(&protocol.EventSourcedInit{
			ServiceName: sn,
			EntityId:    "e3",
		}))
		if err != nil {
			t.Fatal(err)
		}

		// add line item
		addLineItem := &shoppingcart.AddLineItem{UserId: "user1", ProductId: "e-bike-1", Name: "e-Bike", Quantity: 2}
		r, err := sendRecvCmd(
			&protocol.Command{EntityId: "e3", Id: cmdId, Name: "AddLineItem"},
			addLineItem,
			handle,
		)
		if err != nil {
			t.Fatal(err)
		}
		switch m := r.Message.(type) {
		case *protocol.EventSourcedStreamOut_Reply:
			checkCommandId(cmdId, m, t)
			t.Run("the Reply should have events", func(t *testing.T) {
				events := m.Reply.GetEvents()
				if got, want := len(events), 1; got != want {
					t.Fatalf("len(events) = %d; want %d", got, want)
				}
				itemAdded := &domain.ItemAdded{}
				err := encoding.UnmarshalAny(events[0], itemAdded)
				if err != nil {
					t.Fatal(err)
				}
				if got, want := itemAdded.Item.ProductId, addLineItem.ProductId; got != want {
					t.Fatalf("itemAdded.Item.ProductId = %s; want; %s", got, want)
				}
				if got, want := itemAdded.Item.Name, addLineItem.Name; got != want {
					t.Fatalf("itemAdded.Item.Name = %s; want; %s", got, want)
				}
				if got, want := itemAdded.Item.Quantity, addLineItem.Quantity; got != want {
					t.Fatalf("itemAdded.Item.Quantity = %d; want; %d", got, want)
				}
			})
			t.Run("the Reply should have a snapshot", func(t *testing.T) {
				snapshot := m.Reply.GetSnapshot()
				if snapshot == nil {
					t.Fatalf("snapshot was nil but should not")
				}
				cart := &domain.Cart{}
				err := encoding.UnmarshalAny(snapshot, cart)
				if err != nil {
					t.Fatal(err)
				}
				if got, want := len(cart.Items), 1; got != want {
					t.Fatalf("len(cart.Items) = %d; want: %d", got, want)
				}
				item := cart.Items[0]
				if got, want := item.ProductId, addLineItem.ProductId; got != want {
					t.Fatalf("itemAdded.Item.ProductId = %s; want; %s", got, want)
				}
				if got, want := item.Name, addLineItem.Name; got != want {
					t.Fatalf("itemAdded.Item.Name = %s; want; %s", got, want)
				}
				if got, want := item.Quantity, addLineItem.Quantity; got != want {
					t.Fatalf("itemAdded.Item.Quantity = %d; want; %d", got, want)
				}
			})
		case *protocol.EventSourcedStreamOut_Failure:
			checkCommandId(cmdId, m, t)
			t.Fatalf("expected reply but got: %+v", m.Failure)
		default:
			t.Fatalf("unexpected message: %+v", m)
		}

		// get the shopping cart
		cmdId++
		r, err = sendRecvCmd(
			&protocol.Command{
				EntityId: "e2",
				Id:       cmdId,
				Name:     "GetShoppingCart",
			},
			&shoppingcart.GetShoppingCart{UserId: "user1"},
			handle,
		)
		if err != nil {
			t.Fatal(err)
		}
		switch m := r.Message.(type) {
		case *protocol.EventSourcedStreamOut_Reply:
			checkCommandId(cmdId, m, t)
			payload := m.Reply.GetClientAction().GetReply().GetPayload()
			cart := &shoppingcart.Cart{}
			if err := encoding.UnmarshalAny(payload, cart); err != nil {
				t.Fatal(err)
			}
			if l := len(cart.Items); l != 1 {
				t.Fatalf("len(cart.Items) is: %d, but should be 1", l)
			}
			li, ai := cart.Items[0], addLineItem
			if got, want := li.Quantity, ai.Quantity; got != want {
				t.Fatalf("Quantity = %d; want: %d", got, want)
			}
			if got, want := li.Name, ai.Name; got != want {
				t.Fatalf("Name = %s; want: %s", got, want)
			}
			if got, want := li.ProductId, ai.ProductId; got != want {
				t.Fatalf("ProductId = %s; want: %s", got, want)
			}
		case *protocol.EventSourcedStreamOut_Failure:
			t.Fatalf("expected reply but got: %+v", m.Failure)
		default:
			t.Fatalf("unexpected message: %+v", m)
		}
	})

	cmdId++
	t.Run("A init message with an initial snapshot should initialise an entity", func(t *testing.T) {
		esc := protocol.NewEventSourcedClient(conn)
		handle, err := esc.Handle(ctx)
		if err != nil {
			t.Fatal(err)
		}
		cart := &domain.Cart{Items: make([]*domain.LineItem, 0)}
		cart.Items = append(cart.Items, &domain.LineItem{
			ProductId: "e-bike-2",
			Name:      "Cross",
			Quantity:  3,
		})
		cart.Items = append(cart.Items, &domain.LineItem{
			ProductId: "e-bike-3",
			Name:      "Cross TWO",
			Quantity:  1,
		})
		cart.Items = append(cart.Items, &domain.LineItem{
			ProductId: "e-bike-4",
			Name:      "City",
			Quantity:  5,
		})
		any, err := encoding.MarshalAny(cart)
		if err != nil {
			t.Fatal(err)
		}
		err = handle.Send(initMsg(&protocol.EventSourcedInit{
			ServiceName: sn,
			EntityId:    "e9",
			Snapshot: &protocol.EventSourcedSnapshot{
				SnapshotSequence: 0,
				Snapshot:         any,
			},
		}))
		r, err := sendRecvCmd(
			&protocol.Command{
				EntityId: "e9",
				Id:       cmdId,
				Name:     "GetShoppingCart",
			},
			&shoppingcart.GetShoppingCart{UserId: "user1"},
			handle,
		)
		if err != nil {
			t.Fatal(err)
		}
		switch m := r.Message.(type) {
		case *protocol.EventSourcedStreamOut_Failure:
			t.Fatalf("expected reply but got: %+v", m.Failure)
		case *protocol.EventSourcedStreamOut_Reply:
			checkCommandId(cmdId, m, t)
			reply := &domain.Cart{}
			err := encoding.UnmarshalAny(m.Reply.GetClientAction().GetReply().GetPayload(), reply)
			if err != nil {
				t.Fatal(err)
			}
			if l := len(reply.Items); l != 3 {
				t.Fatalf("len(cart.Items) is: %d, but should be 1", l)
			}
			for i := 0; i < len(reply.Items); i++ {
				li := reply.Items[i]
				ci := cart.Items[i]
				if got, want := li.Quantity, ci.Quantity; got != want {
					t.Fatalf("Quantity = %d; want: %d", got, want)
				}
				if got, want := li.Name, ci.Name; got != want {
					t.Fatalf("Name = %s; want: %s", got, want)
				}
				if got, want := li.ProductId, ci.ProductId; got != want {
					t.Fatalf("ProductId = %s; want: %s", got, want)
				}
			}
		default:
			t.Fatalf("unexpected message: %+v", m)
		}
	})

	cmdId++
	t.Run("An initialised entity with an event sent should return its applied state", func(t *testing.T) {
		esc := protocol.NewEventSourcedClient(conn)
		handle, err := esc.Handle(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// init
		err = handle.Send(initMsg(&protocol.EventSourcedInit{
			ServiceName: sn,
			EntityId:    "e20",
		}))
		// send an event
		lineItem := &domain.LineItem{
			ProductId: "e-bike-100",
			Name:      "AMP 100",
			Quantity:  1,
		}
		event, err := encoding.MarshalAny(&domain.ItemAdded{Item: lineItem})
		if err != nil {
			t.Fatal(err)
		}
		err = handle.Send(eventMsg(&protocol.EventSourcedEvent{Sequence: 0, Payload: event}))
		if err != nil {
			t.Fatal(err)
		}
		r, err := sendRecvCmd(
			&protocol.Command{
				EntityId: "e20",
				Id:       cmdId,
				Name:     "GetShoppingCart",
			},
			&shoppingcart.GetShoppingCart{UserId: "user1"},
			handle,
		)
		if err != nil {
			t.Fatal(err)
		}
		switch m := r.Message.(type) {
		case *protocol.EventSourcedStreamOut_Reply:
			// this is what we expect
			checkCommandId(cmdId, m, t)
			payload := m.Reply.GetClientAction().GetReply().GetPayload()
			cart := &shoppingcart.Cart{}
			if err := encoding.UnmarshalAny(payload, cart); err != nil {
				t.Fatal(err)
			}
			if l := len(cart.Items); l != 1 {
				t.Fatalf("len(cart.Items) is: %d, but should be 1", l)
			}
			li := cart.Items[0]
			if got, want := li.Quantity, lineItem.Quantity; got != want {
				t.Fatalf("Quantity = %d; want: %d", got, want)
			}
			if got, want := li.Name, lineItem.Name; got != want {
				t.Fatalf("Name = %s; want: %s", got, want)
			}
			if got, want := li.ProductId, lineItem.ProductId; got != want {
				t.Fatalf("ProductId = %s; want: %s", got, want)
			}
		case *protocol.EventSourcedStreamOut_Failure:
			t.Fatalf("expected reply but got: %+v", m.Failure)
		default:
			t.Fatalf("unexpected message: %+v", m)
		}

	})

	t.Run("Adding negative quantity should fail", func(t *testing.T) {
		esc := protocol.NewEventSourcedClient(conn)
		handle, err := esc.Handle(ctx)
		if err != nil {
			t.Fatal(err)
		}
		entityId := "e23"
		err = handle.Send(initMsg(&protocol.EventSourcedInit{
			ServiceName: sn,
			EntityId:    entityId,
		}))
		if err != nil {
			t.Fatal(err)
		}
		// add line item
		add := []struct {
			command *protocol.Command
			payload interface{}
		}{
			{
				&protocol.Command{EntityId: entityId, Name: "AddLineItem"},
				&shoppingcart.AddLineItem{UserId: "user1", ProductId: "e-bike-1", Name: "e-Bike", Quantity: 1},
			},
			{
				&protocol.Command{EntityId: entityId, Name: "AddLineItem"},
				&shoppingcart.AddLineItem{UserId: "user1", ProductId: "e-bike-2", Name: "e-Bike 2", Quantity: 2},
			},
			{
				&protocol.Command{EntityId: entityId, Name: "AddLineItem"},
				&shoppingcart.AddLineItem{UserId: "user1", ProductId: "e-bike-2", Name: "e-Bike 2", Quantity: -1},
			},
		}
		for i, a := range add {
			a.command.Id = cmdId
			r, err := sendRecvCmd(a.command, a.payload, handle)
			if err != nil {
				t.Fatal(err)
			}
			switch m := r.Message.(type) {
			case *protocol.EventSourcedStreamOut_Reply:
				checkCommandId(cmdId, m, t)
				if i < 2 && m.Reply.ClientAction.GetFailure() != nil {
					t.Fatalf("unexpected ClientAction failure: %+v", m)
				}
				if i == 2 && m.Reply.ClientAction.GetFailure() == nil {
					t.Fatalf("expected ClientAction failure: %+v", m)
				}
			case *protocol.EventSourcedStreamOut_Failure:
				checkCommandId(cmdId, m, t)
				t.Fatalf("expected reply but got: %+v", m.Failure)
			default:
				t.Fatalf("unexpected message: %+v", m)
			}
			cmdId++
		}
	})

	t.Run("Removing a non existent item should fail", func(t *testing.T) {
		esc := protocol.NewEventSourcedClient(conn)
		handle, err := esc.Handle(ctx)
		if err != nil {
			t.Fatal(err)
		}
		entityId := "e24"
		err = handle.Send(initMsg(&protocol.EventSourcedInit{
			ServiceName: sn,
			EntityId:    entityId,
		}))
		if err != nil {
			t.Fatal(err)
		}
		// add line item
		add := []struct {
			command *protocol.Command
			payload interface{}
		}{
			{
				&protocol.Command{EntityId: entityId, Name: "AddLineItem"},
				&shoppingcart.AddLineItem{UserId: "user1", ProductId: "e-bike-1", Name: "e-Bike", Quantity: 1},
			},
			{
				&protocol.Command{EntityId: entityId, Name: "AddLineItem"},
				&shoppingcart.AddLineItem{UserId: "user1", ProductId: "e-bike-2", Name: "e-Bike 2", Quantity: 2},
			},
			{
				&protocol.Command{EntityId: entityId, Name: "RemoveLineItem"},
				&shoppingcart.RemoveLineItem{UserId: "user1", ProductId: "e-bike-1"},
			},
			{
				&protocol.Command{EntityId: entityId, Name: "RemoveLineItem"},
				&shoppingcart.RemoveLineItem{UserId: "user1", ProductId: "e-bike-1"},
			},
		}
		for i, a := range add {
			a.command.Id = cmdId
			r, err := sendRecvCmd(a.command, a.payload, handle)
			if err != nil {
				t.Fatal(err)
			}
			switch m := r.Message.(type) {
			case *protocol.EventSourcedStreamOut_Reply:
				checkCommandId(cmdId, m, t)
				if i <= 2 && m.Reply.ClientAction.GetFailure() != nil {
					t.Fatalf("unexpected ClientAction failure: %+v", m)
				}
				if i == 3 && m.Reply.ClientAction.GetFailure() == nil {
					t.Fatalf("expected ClientAction failure: %+v", m)
				}
			case *protocol.EventSourcedStreamOut_Failure:
				checkCommandId(cmdId, m, t)
				t.Fatalf("expected reply but got: %+v", m.Failure)
			default:
				t.Fatalf("unexpected message: %+v", m)
			}
			cmdId++
		}
	})

	cmdId++
	t.Run("Adding and Removing LineItems", func(t *testing.T) {
		esc := protocol.NewEventSourcedClient(conn)
		handle, err := esc.Handle(ctx)
		if err != nil {
			t.Fatal(err)
		}
		entityId := "e22"
		err = handle.Send(initMsg(&protocol.EventSourcedInit{
			ServiceName: sn,
			EntityId:    entityId,
		}))
		if err != nil {
			t.Fatal(err)
		}
		// add line item
		add := []struct {
			command *protocol.Command
			payload interface{}
		}{
			{
				&protocol.Command{EntityId: entityId, Name: "AddLineItem"},
				&shoppingcart.AddLineItem{UserId: "user1", ProductId: "e-bike-1", Name: "e-Bike", Quantity: 1},
			},
			{
				&protocol.Command{EntityId: entityId, Name: "AddLineItem"},
				&shoppingcart.AddLineItem{UserId: "user1", ProductId: "e-bike-2", Name: "e-Bike 2", Quantity: 2},
			},
			{
				&protocol.Command{EntityId: entityId, Name: "AddLineItem"},
				&shoppingcart.AddLineItem{UserId: "user1", ProductId: "e-bike-3", Name: "e-Bike 3", Quantity: 3},
			},
			{
				&protocol.Command{EntityId: entityId, Name: "AddLineItem"},
				&shoppingcart.AddLineItem{UserId: "user1", ProductId: "e-bike-3", Name: "e-Bike 3", Quantity: 4},
			},
		}
		for _, a := range add {
			a.command.Id = cmdId
			r, err := sendRecvCmd(a.command, a.payload, handle)
			if err != nil {
				t.Fatal(err)
			}
			switch m := r.Message.(type) {
			case *protocol.EventSourcedStreamOut_Reply:
				checkCommandId(cmdId, m, t)
				if m.Reply.ClientAction.GetFailure() != nil {
					t.Fatalf("unexpected ClientAction failure: %+v", m)
				}
			case *protocol.EventSourcedStreamOut_Failure:
				checkCommandId(cmdId, m, t)
				t.Fatalf("expected reply but got: %+v", m.Failure)
			default:
				t.Fatalf("unexpected message: %+v", m)
			}
			cmdId++
		}

		// get the shopping cart
		cmdId++
		r, err := sendRecvCmd(
			&protocol.Command{
				EntityId: entityId,
				Id:       cmdId,
				Name:     "GetShoppingCart",
			},
			&shoppingcart.GetShoppingCart{UserId: "user1"},
			handle,
		)
		if err != nil {
			t.Fatal(err)
		}
		switch m := r.Message.(type) {
		case *protocol.EventSourcedStreamOut_Reply:
			checkCommandId(cmdId, m, t)
			payload := m.Reply.GetClientAction().GetReply().GetPayload()
			cart := &shoppingcart.Cart{}
			if err := encoding.UnmarshalAny(payload, cart); err != nil {
				t.Fatal(err)
			}
			if got, want := len(cart.Items), 3; got != want {
				t.Fatalf("len(cart.Items) = %d; want: %d ", got, want)
			}
			eBike3Count := int32(0)
			for _, c := range cart.Items {
				if c.ProductId == "e-bike-3" {
					eBike3Count = eBike3Count + c.Quantity
				}
			}
			if got, want := eBike3Count, int32(7); got != want {
				t.Fatalf("eBike3Count = %d; want: %d ", got, want)
			}
		case *protocol.EventSourcedStreamOut_Failure:
			t.Fatalf("expected reply but got: %+v", m.Failure)
		default:
			t.Fatalf("unexpected message: %+v", m)
		}

		// remove item
		remove := []struct {
			command *protocol.Command
			payload interface{}
		}{
			{
				&protocol.Command{EntityId: entityId, Name: "RemoveLineItem"},
				&shoppingcart.RemoveLineItem{UserId: "user1", ProductId: "e-bike-1"},
			},
			{
				&protocol.Command{EntityId: entityId, Name: "RemoveLineItem"},
				&shoppingcart.RemoveLineItem{UserId: "user1", ProductId: "e-bike-2"},
			},
			{
				&protocol.Command{EntityId: entityId, Name: "RemoveLineItem"},
				&shoppingcart.RemoveLineItem{UserId: "user1", ProductId: "e-bike-3"},
			},
		}
		for _, r := range remove {
			r.command.Id = cmdId
			r, err := sendRecvCmd(r.command, r.payload, handle)
			if err != nil {
				t.Fatal(err)
			}
			switch m := r.Message.(type) {
			case *protocol.EventSourcedStreamOut_Reply:
				checkCommandId(cmdId, m, t)
				if m.Reply.ClientAction.GetFailure() != nil {
					t.Fatalf("unexpected ClientAction failure: %+v", m)
				}
			case *protocol.EventSourcedStreamOut_Failure:
				checkCommandId(cmdId, m, t)
				t.Fatalf("expected reply but got: %+v", m.Failure)
			default:
				t.Fatalf("unexpected message: %+v", m)
			}
			cmdId++
		}

		// get the shopping cart
		cmdId++
		r, err = sendRecvCmd(
			&protocol.Command{EntityId: entityId, Id: cmdId, Name: "GetShoppingCart"},
			&shoppingcart.GetShoppingCart{UserId: "user1"},
			handle,
		)
		if err != nil {
			t.Fatal(err)
		}
		switch m := r.Message.(type) {
		case *protocol.EventSourcedStreamOut_Reply:
			checkCommandId(cmdId, m, t)
			payload := m.Reply.GetClientAction().GetReply().GetPayload()
			cart := &shoppingcart.Cart{}
			if err := encoding.UnmarshalAny(payload, cart); err != nil {
				t.Fatal(err)
			}
			if got, want := len(cart.Items), 0; got != want {
				t.Fatalf("len(cart.Items) = %d; want: %d ", got, want)
			}
		case *protocol.EventSourcedStreamOut_Failure:
			t.Fatalf("expected reply but got: %+v", m.Failure)
		default:
			t.Fatalf("unexpected message: %+v", m)
		}
	})

	t.Run("Send the User Function an ErrorMessage", func(t *testing.T) {
		reportError, err := edc.ReportError(ctx, &protocol.UserFunctionError{Message: "an error occured"})
		if err != nil {
			t.Fatal(err)
		}
		if reportError == nil {
			t.Fatalf("reportError was nil")
		}
	})
}

func checkCommandId(cmdId int64, m interface{}, t *testing.T) {
	switch m := m.(type) {
	case *protocol.EventSourcedStreamOut_Reply:
		if got, want := m.Reply.CommandId, cmdId; got != want {
			t.Fatalf("expected command id: %v wanted: %d", m.Reply.CommandId, cmdId)
		}
	case *protocol.EventSourcedStreamOut_Failure:
		if got, want := m.Failure.CommandId, cmdId; got != want {
			t.Fatalf("expected command id: %v wanted: %d", m.Failure.CommandId, cmdId)
		}
	default:
		t.Fatalf("unexpected message: %+v", m)
	}
}

func sendRecvCmd(c *protocol.Command, x interface{}, handle protocol.EventSourced_HandleClient) (*protocol.EventSourcedStreamOut, error) {
	any, err := encoding.MarshalAny(x)
	if err != nil {
		return nil, err
	}
	c.Payload = any
	err = handle.Send(commandMsg(c))
	if err != nil {
		return nil, err
	}
	return handle.Recv()
}

func eventMsg(e *protocol.EventSourcedEvent) *protocol.EventSourcedStreamIn {
	return &protocol.EventSourcedStreamIn{
		Message: &protocol.EventSourcedStreamIn_Event{
			Event: e,
		},
	}
}

func initMsg(i *protocol.EventSourcedInit) *protocol.EventSourcedStreamIn {
	return &protocol.EventSourcedStreamIn{
		Message: &protocol.EventSourcedStreamIn_Init{
			Init: i,
		},
	}
}

func commandMsg(c *protocol.Command) *protocol.EventSourcedStreamIn {
	return &protocol.EventSourcedStreamIn{
		Message: &protocol.EventSourcedStreamIn_Command{
			Command: c,
		},
	}
}
