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
	tc "github.com/cloudstateio/go-support/tck/proto/crdt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/empty"
)

type SyntheticCRDTs struct {
	id          crdt.EntityId
	gCounter    *crdt.GCounter
	pnCounter   *crdt.PNCounter
	gSet        *crdt.GSet
	orSet       *crdt.ORSet
	flag        *crdt.Flag
	lwwRegister *crdt.LWWRegister
	vote        *crdt.Vote
}

func NewEntity(id crdt.EntityId) *SyntheticCRDTs {
	return &SyntheticCRDTs{id: id}
}

func (s *SyntheticCRDTs) Set(_ *crdt.Context, c crdt.CRDT) {
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
	}
}

func (s *SyntheticCRDTs) Default(c *crdt.Context) (crdt.CRDT, error) {
	if strings.HasPrefix(c.EntityId.String(), "gcounter-") {
		return crdt.NewGCounter(), nil
	}
	if strings.HasPrefix(c.EntityId.String(), "pncounter-") {
		return crdt.NewPNCounter(), nil
	}
	if strings.HasPrefix(c.EntityId.String(), "gset-") {
		return crdt.NewGSet(), nil
	}
	if strings.HasPrefix(c.EntityId.String(), "orset-") {
		return crdt.NewORSet(), nil
	}
	if strings.HasPrefix(c.EntityId.String(), "flag-") {
		return crdt.NewFlag(), nil
	}
	if strings.HasPrefix(c.EntityId.String(), "vote-") {
		return crdt.NewVote(), nil
	}
	if strings.HasPrefix(c.EntityId.String(), "lwwregister-") {
		return crdt.NewLWWRegister(nil), nil
	}
	return nil, errors.New("unknown entity type")
}

func (s *SyntheticCRDTs) HandleCommand(cc *crdt.CommandContext, name string, cmd proto.Message) (*any.Any, error) {
	switch c := cmd.(type) {
	case *tc.GCounterRequest:
		for _, as := range c.GetActions() {
			switch a := as.Action.(type) {
			case *tc.GCounterRequestAction_Increment:
				s.gCounter.Increment(a.Increment.GetValue())
				pb := &tc.GCounterValue{Value: s.gCounter.Value()}
				if a.Increment.FailWith != "" {
					return nil, errors.New(a.Increment.FailWith)
				}
				return encoding.MarshalAny(pb)
			case *tc.GCounterRequestAction_Get:
				if a.Get.FailWith != "" {
					return nil, errors.New(a.Get.FailWith)
				}
				return encoding.MarshalAny(&tc.GCounterValue{Value: s.gCounter.Value()})
			case *tc.GCounterRequestAction_Delete:
				cc.Delete()
				if a.Delete.FailWith != "" {
					return nil, errors.New(a.Delete.FailWith)
				}
				return encoding.MarshalAny(&empty.Empty{})
			}
		}
	case *tc.PNCounterRequest:
		for _, as := range c.GetActions() {
			switch a := as.Action.(type) {
			case *tc.PNCounterRequestAction_Increment:
				s.pnCounter.Increment(a.Increment.Value)
				if a.Increment.FailWith != "" {
					return nil, errors.New(a.Increment.FailWith)
				}
				return encoding.MarshalAny(&tc.PNCounterResponse{Value: &tc.PNCounterValue{Value: s.pnCounter.Value()}})
			case *tc.PNCounterRequestAction_Decrement:
				s.pnCounter.Decrement(a.Decrement.Value)
				if a.Decrement.FailWith != "" {
					return nil, errors.New(a.Decrement.FailWith)
				}
				return encoding.MarshalAny(&tc.PNCounterResponse{Value: &tc.PNCounterValue{Value: s.pnCounter.Value()}})
			case *tc.PNCounterRequestAction_Get:
				if a.Get.FailWith != "" {
					return nil, errors.New(a.Get.FailWith)
				}
				return encoding.MarshalAny(&tc.PNCounterResponse{Value: &tc.PNCounterValue{Value: s.pnCounter.Value()}})
			case *tc.PNCounterRequestAction_Delete:
				cc.Delete()
				if a.Delete.FailWith != "" {
					return nil, errors.New(a.Delete.FailWith)
				}
				return encoding.MarshalAny(&empty.Empty{})
			}
		}
	case *tc.GSetRequest:
		for _, as := range c.GetActions() {
			switch a := as.Action.(type) {
			case *tc.GSetRequestAction_Get:
				v := tc.GSetValueAnySupport{Values: make([]*tc.AnySupportType, 0, len(s.gSet.Value()))}
				for _, a := range s.gSet.Value() {
					v.Values = append(v.Values, asAnySupportType(a))
				}
				if a.Get.FailWith != "" {
					return nil, errors.New(a.Get.FailWith)
				}
				return encoding.MarshalAny(&v)
			case *tc.GSetRequestAction_Delete:
				cc.Delete()
				if a.Delete.FailWith != "" {
					return nil, errors.New(a.Delete.FailWith)
				}
				return encoding.MarshalAny(&empty.Empty{})
			case *tc.GSetRequestAction_Add:
				anySupportAdd(s.gSet, a.Add.Value)
				v := tc.GSetValueAnySupport{Values: make([]*tc.AnySupportType, 0, len(s.gSet.Value()))}
				for _, a := range s.gSet.Value() {
					if strings.HasPrefix(a.TypeUrl, encoding.JSONTypeURLPrefix) {
						v.Values = append(v.Values, &tc.AnySupportType{
							Value: &tc.AnySupportType_AnyValue{AnyValue: a},
						})
						continue
					}
					v.Values = append(v.Values, asAnySupportType(a))
				}
				if a.Add.FailWith != "" {
					return nil, errors.New(a.Add.FailWith)
				}
				return encoding.MarshalAny(&v)
			}
		}
	case *tc.ORSetRequest:
		for _, as := range c.GetActions() {
			switch a := as.Action.(type) {
			case *tc.ORSetRequestAction_Get:
				if a.Get.FailWith != "" {
					return nil, errors.New(a.Get.FailWith)
				}
				return encoding.MarshalAny(&tc.ORSetValue{Values: s.orSet.Value()})
			case *tc.ORSetRequestAction_Delete:
				cc.Delete()
				if a.Delete.FailWith != "" {
					return nil, errors.New(a.Delete.FailWith)
				}
				return encoding.MarshalAny(&empty.Empty{})
			case *tc.ORSetRequestAction_Add:
				anySupportAdd(s.orSet, a.Add.Value)
				if a.Add.FailWith != "" {
					return nil, errors.New(a.Add.FailWith)
				}
				return encoding.MarshalAny(&tc.ORSetValue{Values: s.orSet.Value()})
			case *tc.ORSetRequestAction_Remove:
				anySupportRemove(s.orSet, a.Remove.Value)
				if a.Remove.FailWith != "" {
					return nil, errors.New(a.Remove.FailWith)
				}
				return encoding.MarshalAny(&tc.ORSetValue{Values: s.orSet.Value()})
			}
		}
	case *tc.FlagRequest:
		for _, as := range c.GetActions() {
			switch a := as.Action.(type) {
			case *tc.FlagRequestAction_Get:
				if a.Get.FailWith != "" {
					return nil, errors.New(a.Get.FailWith)
				}
				return encoding.MarshalAny(&tc.FlagValue{Value: s.flag.Value()})
			case *tc.FlagRequestAction_Delete:
				if a.Delete.FailWith != "" {
					return nil, errors.New(a.Delete.FailWith)
				}
				return encoding.MarshalAny(&empty.Empty{})
			case *tc.FlagRequestAction_Enable:
				if a.Enable.FailWith != "" {
					return nil, errors.New(a.Enable.FailWith)
				}
				s.flag.Enable()
				return encoding.MarshalAny(&tc.FlagValue{Value: s.flag.Value()})
			}
		}
	case *tc.LWWRegisterRequest:
		for _, as := range c.GetActions() {
			switch a := as.Action.(type) {
			case *tc.LWWRegisterRequestAction_Get:
				if a.Get.FailWith != "" {
					return nil, errors.New(a.Get.FailWith)
				}
				return encoding.MarshalAny(&tc.FlagValue{Value: s.flag.Value()})
			case *tc.LWWRegisterRequestAction_Delete:
				if a.Delete.FailWith != "" {
					return nil, errors.New(a.Delete.FailWith)
				}
				return encoding.MarshalAny(&empty.Empty{})
			case *tc.LWWRegisterRequestAction_Set:
				if a.Set.FailWith != "" {
					return nil, errors.New(a.Set.FailWith)
				}
				anySupportAdd(&anySupportAdderSetter{s.lwwRegister}, a.Set.GetValue())
				return encoding.MarshalAny(&tc.LWWRegisterValue{Value: s.lwwRegister.Value()})
			case *tc.LWWRegisterRequestAction_SetWithClock:
				if a.SetWithClock.FailWith != "" {
					return nil, errors.New(a.SetWithClock.FailWith)
				}
				anySupportSetClock(
					s.lwwRegister,
					a.SetWithClock.GetValue(),
					crdt.Clock(uint64(a.SetWithClock.GetClock().Number())),
					a.SetWithClock.CustomClockValue,
				)
				return encoding.MarshalAny(&tc.LWWRegisterValue{Value: s.lwwRegister.Value()})
			}
		}
	}
	return nil, errors.New("unhandled command")
}

func asAnySupportType(x *any.Any) *tc.AnySupportType {
	switch x.TypeUrl {
	case encoding.PrimitiveTypeURLPrefixBool:
		return &tc.AnySupportType{
			Value: &tc.AnySupportType_BoolValue{BoolValue: encoding.DecodeBool(x)},
		}
	case encoding.PrimitiveTypeURLPrefixBytes:
		return &tc.AnySupportType{
			Value: &tc.AnySupportType_BytesValue{BytesValue: encoding.DecodeBytes(x)},
		}
	case encoding.PrimitiveTypeURLPrefixFloat:
		return &tc.AnySupportType{
			Value: &tc.AnySupportType_FloatValue{FloatValue: encoding.DecodeFloat32(x)},
		}
	case encoding.PrimitiveTypeURLPrefixDouble:
		return &tc.AnySupportType{
			Value: &tc.AnySupportType_DoubleValue{DoubleValue: encoding.DecodeFloat64(x)},
		}
	case encoding.PrimitiveTypeURLPrefixInt32:
		return &tc.AnySupportType{
			Value: &tc.AnySupportType_Int32Value{Int32Value: encoding.DecodeInt32(x)},
		}
	case encoding.PrimitiveTypeURLPrefixInt64:
		return &tc.AnySupportType{
			Value: &tc.AnySupportType_Int64Value{Int64Value: encoding.DecodeInt64(x)},
		}
	case encoding.PrimitiveTypeURLPrefixString:
		return &tc.AnySupportType{
			Value: &tc.AnySupportType_StringValue{StringValue: encoding.DecodeString(x)},
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

func anySupportRemove(r anySupportRemover, t *tc.AnySupportType) {
	switch v := t.Value.(type) {
	case *tc.AnySupportType_AnyValue:
		r.Remove(v.AnyValue)
	case *tc.AnySupportType_StringValue:
		r.Remove(encoding.String(v.StringValue))
	case *tc.AnySupportType_BytesValue:
		r.Remove(encoding.Bytes(v.BytesValue))
	case *tc.AnySupportType_BoolValue:
		r.Remove(encoding.Bool(v.BoolValue))
	case *tc.AnySupportType_DoubleValue:
		r.Remove(encoding.Float64(v.DoubleValue))
	case *tc.AnySupportType_FloatValue:
		r.Remove(encoding.Float32(v.FloatValue))
	case *tc.AnySupportType_Int32Value:
		r.Remove(encoding.Int32(v.Int32Value))
	case *tc.AnySupportType_Int64Value:
		r.Remove(encoding.Int64(v.Int64Value))
	}
}

func anySupportAdd(a anySupportAdder, t *tc.AnySupportType) {
	switch v := t.Value.(type) {
	case *tc.AnySupportType_AnyValue:
		a.Add(v.AnyValue)
	case *tc.AnySupportType_StringValue:
		a.Add(encoding.String(v.StringValue))
	case *tc.AnySupportType_BytesValue:
		a.Add(encoding.Bytes(v.BytesValue))
	case *tc.AnySupportType_BoolValue:
		a.Add(encoding.Bool(v.BoolValue))
	case *tc.AnySupportType_DoubleValue:
		a.Add(encoding.Float64(v.DoubleValue))
	case *tc.AnySupportType_FloatValue:
		a.Add(encoding.Float32(v.FloatValue))
	case *tc.AnySupportType_Int32Value:
		a.Add(encoding.Int32(v.Int32Value))
	case *tc.AnySupportType_Int64Value:
		a.Add(encoding.Int64(v.Int64Value))
	}
}

func anySupportSetClock(r *crdt.LWWRegister, t *tc.AnySupportType, clock crdt.Clock, customValue int64) {
	switch v := t.Value.(type) {
	case *tc.AnySupportType_AnyValue:
		r.SetWithClock(v.AnyValue, clock, customValue)
	case *tc.AnySupportType_StringValue:
		r.SetWithClock(encoding.String(v.StringValue), clock, customValue)
	case *tc.AnySupportType_BytesValue:
		r.SetWithClock(encoding.Bytes(v.BytesValue), clock, customValue)
	case *tc.AnySupportType_BoolValue:
		r.SetWithClock(encoding.Bool(v.BoolValue), clock, customValue)
	case *tc.AnySupportType_DoubleValue:
		r.SetWithClock(encoding.Float64(v.DoubleValue), clock, customValue)
	case *tc.AnySupportType_FloatValue:
		r.SetWithClock(encoding.Float32(v.FloatValue), clock, customValue)
	case *tc.AnySupportType_Int32Value:
		r.SetWithClock(encoding.Int32(v.Int32Value), clock, customValue)
	case *tc.AnySupportType_Int64Value:
		r.SetWithClock(encoding.Int64(v.Int64Value), clock, customValue)
	}
}
