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

package eventsourced

import (
	"errors"
	"fmt"

	"github.com/cloudstateio/go-support/cloudstate/protocol"
)

// sendFailure sends a given error to the proxy.
//
// If the error is a protocol.ProtocolFailure a corresponding
// commandId is unwrapped.
//
// failure semantics are defined here:
// https://github.com/cloudstateio/cloudstate/pull/119#discussion_r375619440
func sendFailure(e error, s protocol.EventSourced_HandleServer) error {
	pf := &protocol.ProtocolFailure{}
	if errors.As(e, &pf) {
		desc := pf.F.GetDescription()
		if desc == "" {
			if e := errors.Unwrap(e); e != nil {
				desc = e.Error()
			}
		}
		err := s.Send(&protocol.EventSourcedStreamOut{
			Message: &protocol.EventSourcedStreamOut_Failure{
				Failure: &protocol.Failure{
					CommandId:   pf.F.GetCommandId(),
					Description: desc,
				},
			},
		})
		if err != nil {
			return fmt.Errorf("send of EventSourcedStreamOut Failure failed with: %w", err)
		}
		return nil
	}
	// any other failure is sent as a protocol failure
	err := s.Send(&protocol.EventSourcedStreamOut{
		Message: &protocol.EventSourcedStreamOut_Failure{
			Failure: &protocol.Failure{
				Description: e.Error(),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("send of EventSourcedStreamOut.Failure failed with: %w", err)
	}
	return nil
}
