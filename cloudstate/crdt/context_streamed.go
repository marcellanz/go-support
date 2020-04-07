package crdt

import (
	"errors"
	"fmt"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/golang/protobuf/ptypes/any"
)

/**
 * Register an on change callback for this command.
 *
 * <p>The callback will be invoked any time the CRDT changes. The callback may inspect the CRDT,
 * but any attempt to modify the CRDT will be ignored and the CRDT will crash.
 *
 * <p>If the callback returns a value, that value will be sent down the stream. Alternatively, the
 * callback may forward messages to other entities via the passed in {@link SubscriptionContext}.
 * The callback may also emit side effects to other entities via that context.
 *
 * @param subscriber The subscriber callback.
 */
type ChangeFunc func(c *StreamContext) (*any.Any, error)
type CancelFunc func(c *StreamContext) error

type StreamContext struct {
	*Context
	CommandId   CommandId
	change      ChangeFunc
	cancel      CancelFunc
	failed      error
	forward     *protocol.Forward
	sideEffects []*protocol.SideEffect
	ended       bool
}

func (c *StreamContext) ChangeFunc(f ChangeFunc) {
	c.change = f
}

/**
 * TODO: rewrite as Go documentation
 * Register an on cancel callback for this command.
 *
 * <p>This will be invoked if the client initiates a stream cancel. It will not be invoked if the
 * entity cancels the stream itself via {@link SubscriptionContext#endStream()} from an {@link
 * StreamedCommandContext#onChange(Function)} callback.
 *
 * <p>An on cancel callback may update the CRDT, and may emit side effects via the passed in
 * {@link StreamCancelledContext}.
 *
 * @param effect The effect to perform when this stream is cancelled.
 */
func (c *StreamContext) CancelFunc(f CancelFunc) {
	c.cancel = f
}

var ErrFailCalled = errors.New("context failed by context")

func (c *StreamContext) Fail(err error) {
	// TODO: has to be active, has to be not yet failed
	// "fail(…) already previously invoked!"
	// "This context has already forwarded."
	c.failed = fmt.Errorf("failed with %v: %w", err, ErrFailCalled)
}

func (c *StreamContext) End() {
	c.ended = true
}

func (c *StreamContext) Forward(f *protocol.Forward) {
	// TODO: has to ne not yet forwarded... "This context has already forwarded."
	c.forward = f
}

func (c *StreamContext) SideEffect(e *protocol.SideEffect) {
	c.sideEffects = append(c.sideEffects, e)
}

func (c *StreamContext) clearSideEffect() {
	c.sideEffects = make([]*protocol.SideEffect, 0, cap(c.sideEffects)) // TODO: should we decrease that?
}

var ErrStateChanged = errors.New("CRDT change not allowed")

func (c *StreamContext) changed() (reply *any.Any, err error) {
	// spec impl: checkActive()
	reply, err = c.change(c)
	if c.crdt.HasDelta() {
		err = ErrStateChanged
	}
	return
}

/**
 * TODO: rewrite as Go documentation
 * Register an on cancel callback for this command.
 *
 * <p>This will be invoked if the client initiates a stream cancel. It will not be invoked if the
 * entity cancels the stream itself via {@link SubscriptionContext#endStream()} from an {@link
 * StreamedCommandContext#onChange(Function)} callback.
 *
 * <p>An on cancel callback may update the CRDT, and may emit side effects via the passed in
 * {@link StreamCancelledContext}.
 *
 * @param effect The effect to perform when this stream is cancelled.
 */
func (c *StreamContext) cancelled() error {
	// spec impl: checkActive()
	return c.cancel(c)
}

func (c *Context) enableStreamFor(id CommandId) {
	if _, ok := c.streamedCtx[id]; !ok {
		c.streamedCtx[id] = &StreamContext{
			CommandId:   id,
			Context:     c,
			sideEffects: make([]*protocol.SideEffect, 0),
		}
	}
}

func (c *StreamContext) clientActionFor(reply *any.Any) *protocol.ClientAction {
	if c.failed != nil {
		return &protocol.ClientAction{
			Action: &protocol.ClientAction_Failure{
				Failure: &protocol.Failure{
					CommandId:   c.CommandId.Value(),
					Description: c.failed.Error(),
				},
			},
		}
	}
	if reply != nil {
		if c.forward != nil {
			// spec impl: "Both a reply was returned, and a forward message was sent, choose one or the other."
			// TODO notallowed: "This context has already forwarded."
			return nil
		}
		return &protocol.ClientAction{
			Action: &protocol.ClientAction_Reply{
				Reply: &protocol.Reply{
					Payload: reply,
				},
			},
		}
	}
	if c.forward != nil {
		return &protocol.ClientAction{
			Action: &protocol.ClientAction_Forward{
				Forward: c.forward,
			},
		}
	}
	return nil
}

func (c *StreamContext) stateAction() *protocol.CrdtStateAction {
	if c.crdt == nil {
		return nil
	}
	if c.created {
		if c.crdt.HasDelta() {
			c.created = false
			if c.deleted {
				c.crdt = nil
				return nil
			}
			c.crdt.resetDelta()
			return &protocol.CrdtStateAction{
				Action: &protocol.CrdtStateAction_Create{
					Create: c.crdt.State(),
				},
			}
		}
		if c.deleted {
			c.created = false
			c.crdt = nil
		}
		return nil
	}
	if c.deleted {
		return &protocol.CrdtStateAction{
			Action: &protocol.CrdtStateAction_Delete{Delete: &protocol.CrdtDelete{}},
		}
	}
	if c.crdt.HasDelta() {
		delta := c.crdt.Delta()
		c.crdt.resetDelta()
		return &protocol.CrdtStateAction{
			Action: &protocol.CrdtStateAction_Update{Update: delta},
		}
	}
	return nil
}
