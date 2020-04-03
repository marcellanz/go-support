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
type CancelFunc func(c *StreamContext) (*any.Any, error)

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

var ErrFailedCalled = errors.New("context failed by context")

func (c *StreamContext) Fail(err error) {
	// TODO: has to be active, has to be not yet failed
	// "fail(…) already previously invoked!"
	// "This context has already forwarded."
	c.failed = fmt.Errorf("failed with %v: %w", err, ErrFailedCalled)
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

var ErrStateChanged = errors.New("CRDT change not allowed")

func (c *StreamContext) changed() (reply *any.Any, err error) {
	reply, err = c.change(c)
	if c.crdt.HasDelta() {
		// spec: any attempt to modify the CRDT will be ignored and the CRDT will crash.
		err = ErrStateChanged
	}
	return
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
