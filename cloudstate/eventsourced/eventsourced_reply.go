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

package eventsourced

import (
	"errors"
	"fmt"

	"github.com/cloudstateio/go-support/cloudstate/protocol"
)

var ErrSendFailure = errors.New("unable to send a failure message")
var ErrSend = errors.New("unable to send a message")

// sendEventSourcedReply sends a given EventSourcedReply and if it fails, handles the error wrapping
func sendEventSourcedReply(r *protocol.EventSourcedReply, s protocol.EventSourced_HandleServer) error {
	err := s.Send(&protocol.EventSourcedStreamOut{
		Message: &protocol.EventSourcedStreamOut_Reply{
			Reply: r,
		},
	})
	if err != nil {
		return fmt.Errorf("%s, %w", err, ErrSend)
	}
	return err
}
