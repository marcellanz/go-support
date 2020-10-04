package eventsourced

import (
	"errors"

	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/eventsourced"
	tck "github.com/cloudstateio/go-support/tck/pb/eventsourced"
	"github.com/golang/protobuf/proto"
)

type TestModel struct {
	state string
}

func NewTestModel(id eventsourced.EntityId) eventsourced.EntityHandler {
	return &TestModel{}
}

func (m *TestModel) HandleCommand(ctx *eventsourced.Context, name string, cmd proto.Message) (reply proto.Message, err error) {
	switch c := cmd.(type) {
	case *tck.Request:
		for _, action := range c.GetActions() {
			switch a := action.GetAction().(type) {
			case *tck.RequestAction_Emit:
				ctx.Emit(&tck.Persisted{Value: a.Emit.GetValue()})
			case *tck.RequestAction_Forward:
				any, err := encoding.MarshalAny(&tck.Request{Id: a.Forward.Id})
				if err != nil {
					return nil, err
				}
				ctx.Forward("cloudstate.tck.model.EventSourcedTwo", "Call", any)
			case *tck.RequestAction_Effect:
				req, err := encoding.MarshalAny(&tck.Request{Id: a.Effect.Id})
				if err != nil {
					return nil, err
				}
				ctx.Effect("cloudstate.tck.model.EventSourcedTwo", "Call", req, a.Effect.Synchronous)
			case *tck.RequestAction_Fail:
				return nil, errors.New(a.Fail.GetMessage())
			}
		}
	}
	return &tck.Response{Message: m.state}, nil
}

func (m *TestModel) HandleEvent(ctx *eventsourced.Context, event interface{}) error {
	switch c := event.(type) {
	case *tck.Persisted:
		m.state += c.GetValue()
		return nil
	}
	return errors.New("event not handled")
}

func (m *TestModel) Snapshot(ctx *eventsourced.Context) (snapshot interface{}, err error) {
	return &tck.Persisted{Value: m.state}, nil
}

func (m *TestModel) HandleSnapshot(ctx *eventsourced.Context, snapshot interface{}) error {
	switch s := snapshot.(type) {
	case *tck.Persisted:
		m.state = s.GetValue()
		return nil
	}
	return errors.New("snapshot not handled")
}

type TestModelTwo struct {
}

func (t *TestModelTwo) HandleCommand(ctx *eventsourced.Context, name string, cmd proto.Message) (reply proto.Message, err error) {
	switch cmd.(type) {
	case *tck.Request:
		return &tck.Response{}, nil
	}
	return nil, errors.New("unhandled command: " + name)
}

func (t *TestModelTwo) HandleEvent(ctx *eventsourced.Context, event interface{}) error {
	return nil
}

func NewTestModelTwo(id eventsourced.EntityId) eventsourced.EntityHandler {
	return &TestModelTwo{}
}
