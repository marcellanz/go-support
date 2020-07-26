//
// Copyright 2020 Lightbend Inc.
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

package synth

import (
	"testing"

	"github.com/cloudstateio/go-support/cloudstate/encoding"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/golang/protobuf/proto"
)

type command struct {
	c *protocol.Command
	m proto.Message
}

type state struct {
	s *protocol.CrdtState
}

type delta struct {
	d *protocol.CrdtDelta
}

type proxy struct {
	h               protocol.Crdt_HandleClient
	t               *testing.T
	commandSequence int64
}

func (p *proxy) sendInit(i *protocol.CrdtInit) {
	err := p.h.Send(&protocol.CrdtStreamIn{Message: &protocol.CrdtStreamIn_Init{
		Init: i,
	}})
	if err != nil {
		p.t.Fatal(err)
	}
}

func (p *proxy) sendState(st state) error {
	return p.h.Send(stateMsg(st.s))
}

func (p *proxy) sendRecvState(st state) (*protocol.CrdtStreamOut, error) {
	if err := p.h.Send(stateMsg(st.s)); err != nil {
		return nil, err
	}
	return p.h.Recv()
}

func (p *proxy) sendDelta(d delta) error {
	return p.h.Send(deltaMsg(d.d))
}

func (p *proxy) sendRecvDelta(d delta) (*protocol.CrdtStreamOut, error) {
	if err := p.h.Send(deltaMsg(d.d)); err != nil {
		return nil, err
	}
	return p.h.Recv()
}

func (p *proxy) sendRecvCmd(cmd command) (*protocol.CrdtStreamOut, error) {
	if cmd.c.Id == 0 {
		cmd.c.Id = p.commandSequence
		defer func() { p.commandSequence++ }()
	}
	any, err := encoding.MarshalAny(cmd.m)
	if err != nil {
		return nil, err
	}
	cmd.c.Payload = any
	err = p.h.Send(commandMsg(cmd.c))
	if err != nil {
		return nil, err
	}
	recv, err := p.h.Recv()
	if err != nil {
		return nil, err
	}
	switch recv.Message.(type) {
	case *protocol.CrdtStreamOut_Failure:
	default:
		checkCommandId(p.commandSequence, recv, p.t)
	}
	return recv, err
}

func commandMsg(c *protocol.Command) *protocol.CrdtStreamIn {
	return &protocol.CrdtStreamIn{
		Message: &protocol.CrdtStreamIn_Command{
			Command: c,
		},
	}
}

func stateMsg(s *protocol.CrdtState) *protocol.CrdtStreamIn {
	return &protocol.CrdtStreamIn{
		Message: &protocol.CrdtStreamIn_State{
			State: s,
		},
	}
}

func deltaMsg(d *protocol.CrdtDelta) *protocol.CrdtStreamIn {
	return &protocol.CrdtStreamIn{
		Message: &protocol.CrdtStreamIn_Changed{
			Changed: d,
		},
	}
}

func checkCommandId(cmdId int64, msg interface{}, t *testing.T) {
	switch m := msg.(type) {
	case *protocol.CrdtStreamOut:
		switch out := m.Message.(type) {
		case *protocol.CrdtStreamOut_Reply:
			if got, want := out.Reply.CommandId, cmdId; got != want {
				t.Fatalf("command = %v; wanted: %d, for message:%+v", got, want, out)
			}
		case *protocol.CrdtStreamOut_Failure:
			if got, want := out.Failure.CommandId, cmdId; got != want {
				t.Fatalf("command = %v; wanted: %d, for message:%+v", got, want, out)
			}
		case *protocol.CrdtStreamOut_StreamedMessage:
			if got, want := out.StreamedMessage.CommandId, cmdId; got != want {
				t.Fatalf("command = %v; wanted: %d, for message:%+v", got, want, out)
			}
		case *protocol.CrdtStreamOut_StreamCancelledResponse:
			if got, want := out.StreamCancelledResponse.CommandId, cmdId; got != want {
				t.Fatalf("command = %v; wanted: %d, for message:%+v", got, want, out)
			}
		default:
			t.Fatalf("unexpected message: %+v", m)
		}
	default:
		t.Fatalf("unexpected message: %+v", m)
	}
}
