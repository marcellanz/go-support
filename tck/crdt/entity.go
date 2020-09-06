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

package crdt

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cloudstateio/go-support/cloudstate/crdt"
	"github.com/cloudstateio/go-support/cloudstate/encoding"
	tck "github.com/cloudstateio/go-support/tck/proto/crdt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/empty"
)

type CRDTModel struct {
	id          crdt.EntityId
	gCounter    *crdt.GCounter
	pnCounter   *crdt.PNCounter
	gSet        *crdt.GSet
	orSet       *crdt.ORSet
	flag        *crdt.Flag
	lwwRegister *crdt.LWWRegister
	vote        *crdt.Vote
	orMap       *crdt.ORMap
}

func NewEntity(id crdt.EntityId) *CRDTModel {
	return &CRDTModel{id: id}
}

func (s *CRDTModel) Set(_ *crdt.Context, c crdt.CRDT) {
	switch v := c.(type) {
	case *crdt.GCounter:
		s.gCounter = v
	case *crdt.PNCounter:
		s.pnCounter = v
	case *crdt.GSet:
		s.gSet = v
	case *crdt.ORSet:
		s.orSet = v
	case *crdt.Flag:
		s.flag = v
	case *crdt.LWWRegister:
		s.lwwRegister = v
	case *crdt.Vote:
		s.vote = v
	case *crdt.ORMap:
		s.orMap = v
	}
}

func (s *CRDTModel) Default(c *crdt.Context) (crdt.CRDT, error) {
	switch strings.Split(c.EntityId.String(), "-")[0] {
	case "gcounter":
		return crdt.NewGCounter(), nil
	case "pncounter":
		return crdt.NewPNCounter(), nil
	case "gset":
		return crdt.NewGSet(), nil
	case "orset":
		return crdt.NewORSet(), nil
	case "flag":
		return crdt.NewFlag(), nil
	case "lwwregister":
		return crdt.NewLWWRegister(nil), nil
	case "vote":
		return crdt.NewVote(), nil
	case "ormap":
		return crdt.NewORMap(), nil
	default:
		return nil, errors.New("unknown entity type")
	}
}

func (s *CRDTModel) HandleCommand(cc *crdt.CommandContext, _ string, cmd proto.Message) (*any.Any, error) {
	switch c := cmd.(type) {
	case *tck.GCounterRequest:
		for _, as := range c.GetActions() {
			switch a := as.Action.(type) {
			case *tck.GCounterRequestAction_Increment:
				s.gCounter.Increment(a.Increment.GetValue())
				v := &tck.GCounterValue{Value: s.gCounter.Value()}
				if with := a.Increment.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.GCounterResponse{Value: v})
			case *tck.GCounterRequestAction_Get:
				if with := a.Get.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.GCounterResponse{Value: &tck.GCounterValue{Value: s.gCounter.Value()}})
			case *tck.GCounterRequestAction_Delete:
				cc.Delete()
				if with := a.Delete.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.GCounterResponse{})
			}
		}
	case *tck.PNCounterRequest:
		for _, as := range c.GetActions() {
			switch a := as.Action.(type) {
			case *tck.PNCounterRequestAction_Increment:
				s.pnCounter.Increment(a.Increment.Value)
				if with := a.Increment.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.PNCounterResponse{Value: &tck.PNCounterValue{Value: s.pnCounter.Value()}})
			case *tck.PNCounterRequestAction_Decrement:
				s.pnCounter.Decrement(a.Decrement.Value)
				if with := a.Decrement.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.PNCounterResponse{Value: &tck.PNCounterValue{Value: s.pnCounter.Value()}})
			case *tck.PNCounterRequestAction_Get:
				if with := a.Get.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.PNCounterResponse{Value: &tck.PNCounterValue{Value: s.pnCounter.Value()}})
			case *tck.PNCounterRequestAction_Delete:
				cc.Delete()
				if with := a.Delete.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.PNCounterResponse{})
			}
		}
	case *tck.GSetRequest:
		for _, as := range c.GetActions() {
			switch a := as.Action.(type) {
			case *tck.GSetRequestAction_Get:
				v := make([]*tck.AnySupportType, 0, len(s.gSet.Value()))
				for _, a := range s.gSet.Value() {
					v = append(v, asAnySupportType(a))
				}
				if with := a.Get.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.GSetResponse{Value: &tck.GSetValue{Values: v}})
			case *tck.GSetRequestAction_Delete:
				cc.Delete()
				if with := a.Delete.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.GSetResponse{})
			case *tck.GSetRequestAction_Add:
				anySupportAdd(s.gSet, a.Add.Value)
				v := make([]*tck.AnySupportType, 0, len(s.gSet.Value()))
				for _, a := range s.gSet.Value() {
					if strings.HasPrefix(a.TypeUrl, encoding.JSONTypeURLPrefix) {
						v = append(v, &tck.AnySupportType{
							Value: &tck.AnySupportType_AnyValue{AnyValue: a},
						})
						continue
					}
					v = append(v, asAnySupportType(a))
				}
				if with := a.Add.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.GSetResponse{Value: &tck.GSetValue{Values: v}})
			}
		}
	case *tck.ORSetRequest:
		for _, as := range c.GetActions() {
			switch a := as.Action.(type) {
			case *tck.ORSetRequestAction_Get:
				if with := a.Get.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.ORSetResponse{Value: &tck.ORSetValue{Values: s.orSet.Value()}})
			case *tck.ORSetRequestAction_Delete:
				cc.Delete()
				if with := a.Delete.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.ORSetResponse{})
			case *tck.ORSetRequestAction_Add:
				anySupportAdd(s.orSet, a.Add.Value)
				if with := a.Add.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.ORSetResponse{Value: &tck.ORSetValue{Values: s.orSet.Value()}})
			case *tck.ORSetRequestAction_Remove:
				anySupportRemove(s.orSet, a.Remove.Value)
				if with := a.Remove.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.ORSetResponse{Value: &tck.ORSetValue{Values: s.orSet.Value()}})
			}
		}
	case *tck.FlagRequest:
		for _, as := range c.GetActions() {
			switch a := as.Action.(type) {
			case *tck.FlagRequestAction_Get:
				if with := a.Get.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.FlagResponse{Value: &tck.FlagValue{Value: s.flag.Value()}})
			case *tck.FlagRequestAction_Delete:
				cc.Delete()
				if with := a.Delete.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.FlagResponse{})
			case *tck.FlagRequestAction_Enable:
				if with := a.Enable.FailWith; with != "" {
					return nil, errors.New(with)
				}
				s.flag.Enable()
				return encoding.MarshalAny(&tck.FlagResponse{Value: &tck.FlagValue{Value: s.flag.Value()}})
			}
		}
	case *tck.LWWRegisterRequest:
		for _, as := range c.GetActions() {
			switch a := as.Action.(type) {
			case *tck.LWWRegisterRequestAction_Get:
				if with := a.Get.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.LWWRegisterResponse{Value: &tck.LWWRegisterValue{Value: s.lwwRegister.Value()}})
			case *tck.LWWRegisterRequestAction_Delete:
				cc.Delete()
				if with := a.Delete.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.FlagResponse{})
			case *tck.LWWRegisterRequestAction_Set:
				if with := a.Set.FailWith; with != "" {
					return nil, errors.New(with)
				}
				anySupportAdd(&anySupportAdderSetter{s.lwwRegister}, a.Set.GetValue())
				return encoding.MarshalAny(&tck.LWWRegisterResponse{Value: &tck.LWWRegisterValue{Value: s.lwwRegister.Value()}})
			case *tck.LWWRegisterRequestAction_SetWithClock:
				if with := a.SetWithClock.FailWith; with != "" {
					return nil, errors.New(with)
				}
				anySupportSetClock(
					s.lwwRegister,
					a.SetWithClock.GetValue(),
					crdt.Clock(uint64(a.SetWithClock.GetClock().Number())),
					a.SetWithClock.CustomClockValue,
				)
				return encoding.MarshalAny(&tck.LWWRegisterResponse{Value: &tck.LWWRegisterValue{Value: s.lwwRegister.Value()}})
			}
		}
	case *tck.VoteRequest:
		for _, as := range c.GetActions() {
			switch a := as.Action.(type) {
			case *tck.VoteRequestAction_Get:
				if with := a.Get.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.VoteResponse{
					SelfVote: s.vote.SelfVote(),
					Voters:   s.vote.Voters(),
					VotesFor: s.vote.VotesFor(),
				})
			case *tck.VoteRequestAction_Delete:
				cc.Delete()
				if with := a.Delete.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&empty.Empty{})
			case *tck.VoteRequestAction_Vote:
				s.vote.Vote(a.Vote.GetValue())
				if with := a.Vote.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.VoteResponse{
					SelfVote: s.vote.SelfVote(),
					Voters:   s.vote.Voters(),
					VotesFor: s.vote.VotesFor(),
				})
			}
		}
	case *tck.ORMapRequest:
		for _, as := range c.GetActions() {
			switch a := as.Action.(type) {
			case *tck.ORMapRequestAction_Get:
				if with := a.Get.FailWith; with != "" {
					return nil, errors.New(with)
				} // TODO: finish
				return encoding.MarshalAny(&tck.ORMapResponse{})
			case *tck.ORMapRequestAction_Delete:
				if with := a.Delete.FailWith; with != "" {
					return nil, errors.New(with)
				}
				cc.Delete()
				return encoding.MarshalAny(&tck.ORMapResponse{})
			case *tck.ORMapRequestAction_SetKey:
				// we reuse this entities implementation to
				// handle requests for a certain CRDT of this
				// ORMaps value.
				m := &CRDTModel{}
				switch t := a.SetKey.GetRequest().(type) {
				case *tck.ORMapSet_GCounterRequest:
					ctx := &crdt.CommandContext{Context: &crdt.Context{EntityId: "gcounter"}}
					key := a.SetKey.EntryKey
					if !s.orMap.HasKey(key) {
						c, err := m.Default(ctx.Context)
						if err != nil {
							return nil, err
						}
						s.orMap.Set(key, c)
					}
					m.Set(ctx.Context, s.orMap.Get(key))
					if _, err := m.HandleCommand(ctx, "", t.GCounterRequest); err != nil {
						return nil, err
					}
					return encoding.MarshalAny(&tck.ORMapResponse{
						Entries: &tck.ORMapEntries{
							Values: append(make([]*tck.ORMapEntry, 0), &tck.ORMapEntry{
								EntryKey: key,
								Value:    encoding.Struct(m.gCounter.State()),
							}),
						},
					})
				case *tck.ORMapSet_PnCounterRequest:
				}
				if with := a.SetKey.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.ORMapResponse{})
			case *tck.ORMapRequestAction_DeleteKey:
				s.orMap.Delete(a.DeleteKey.EntryKey)
				if with := a.DeleteKey.FailWith; with != "" {
					return nil, errors.New(with)
				}
				return encoding.MarshalAny(&tck.ORMapResponse{})
			}
		}
	}
	return nil, errors.New("unhandled command")
}

func asAnySupportType(x *any.Any) *tck.AnySupportType {
	switch x.TypeUrl {
	case encoding.PrimitiveTypeURLPrefixBool:
		return &tck.AnySupportType{
			Value: &tck.AnySupportType_BoolValue{BoolValue: encoding.DecodeBool(x)},
		}
	case encoding.PrimitiveTypeURLPrefixBytes:
		return &tck.AnySupportType{
			Value: &tck.AnySupportType_BytesValue{BytesValue: encoding.DecodeBytes(x)},
		}
	case encoding.PrimitiveTypeURLPrefixFloat:
		return &tck.AnySupportType{
			Value: &tck.AnySupportType_FloatValue{FloatValue: encoding.DecodeFloat32(x)},
		}
	case encoding.PrimitiveTypeURLPrefixDouble:
		return &tck.AnySupportType{
			Value: &tck.AnySupportType_DoubleValue{DoubleValue: encoding.DecodeFloat64(x)},
		}
	case encoding.PrimitiveTypeURLPrefixInt32:
		return &tck.AnySupportType{
			Value: &tck.AnySupportType_Int32Value{Int32Value: encoding.DecodeInt32(x)},
		}
	case encoding.PrimitiveTypeURLPrefixInt64:
		return &tck.AnySupportType{
			Value: &tck.AnySupportType_Int64Value{Int64Value: encoding.DecodeInt64(x)},
		}
	case encoding.PrimitiveTypeURLPrefixString:
		return &tck.AnySupportType{
			Value: &tck.AnySupportType_StringValue{StringValue: encoding.DecodeString(x)},
		}
	}
	panic(fmt.Sprintf("no mapping found for TypeUrl: %v", x.TypeUrl)) // we're allowed to panic here :)
}

type anySupportAdder interface {
	Add(x *any.Any)
}

type anySupportSetter interface {
	Set(x *any.Any)
}

type anySupportAdderSetter struct {
	anySupportSetter
}

func (s *anySupportAdderSetter) Add(x *any.Any) {
	s.Set(x)
}

type anySupportRemover interface {
	Remove(x *any.Any)
}

func anySupportRemove(r anySupportRemover, t *tck.AnySupportType) {
	switch v := t.Value.(type) {
	case *tck.AnySupportType_AnyValue:
		r.Remove(v.AnyValue)
	case *tck.AnySupportType_StringValue:
		r.Remove(encoding.String(v.StringValue))
	case *tck.AnySupportType_BytesValue:
		r.Remove(encoding.Bytes(v.BytesValue))
	case *tck.AnySupportType_BoolValue:
		r.Remove(encoding.Bool(v.BoolValue))
	case *tck.AnySupportType_DoubleValue:
		r.Remove(encoding.Float64(v.DoubleValue))
	case *tck.AnySupportType_FloatValue:
		r.Remove(encoding.Float32(v.FloatValue))
	case *tck.AnySupportType_Int32Value:
		r.Remove(encoding.Int32(v.Int32Value))
	case *tck.AnySupportType_Int64Value:
		r.Remove(encoding.Int64(v.Int64Value))
	}
}

func anySupportAdd(a anySupportAdder, t *tck.AnySupportType) {
	switch v := t.Value.(type) {
	case *tck.AnySupportType_AnyValue:
		a.Add(v.AnyValue)
	case *tck.AnySupportType_StringValue:
		a.Add(encoding.String(v.StringValue))
	case *tck.AnySupportType_BytesValue:
		a.Add(encoding.Bytes(v.BytesValue))
	case *tck.AnySupportType_BoolValue:
		a.Add(encoding.Bool(v.BoolValue))
	case *tck.AnySupportType_DoubleValue:
		a.Add(encoding.Float64(v.DoubleValue))
	case *tck.AnySupportType_FloatValue:
		a.Add(encoding.Float32(v.FloatValue))
	case *tck.AnySupportType_Int32Value:
		a.Add(encoding.Int32(v.Int32Value))
	case *tck.AnySupportType_Int64Value:
		a.Add(encoding.Int64(v.Int64Value))
	}
}

func anySupportSetClock(r *crdt.LWWRegister, t *tck.AnySupportType, clock crdt.Clock, customValue int64) {
	switch v := t.Value.(type) {
	case *tck.AnySupportType_AnyValue:
		r.SetWithClock(v.AnyValue, clock, customValue)
	case *tck.AnySupportType_StringValue:
		r.SetWithClock(encoding.String(v.StringValue), clock, customValue)
	case *tck.AnySupportType_BytesValue:
		r.SetWithClock(encoding.Bytes(v.BytesValue), clock, customValue)
	case *tck.AnySupportType_BoolValue:
		r.SetWithClock(encoding.Bool(v.BoolValue), clock, customValue)
	case *tck.AnySupportType_DoubleValue:
		r.SetWithClock(encoding.Float64(v.DoubleValue), clock, customValue)
	case *tck.AnySupportType_FloatValue:
		r.SetWithClock(encoding.Float32(v.FloatValue), clock, customValue)
	case *tck.AnySupportType_Int32Value:
		r.SetWithClock(encoding.Int32(v.Int32Value), clock, customValue)
	case *tck.AnySupportType_Int64Value:
		r.SetWithClock(encoding.Int64(v.Int64Value), clock, customValue)
	}
}
