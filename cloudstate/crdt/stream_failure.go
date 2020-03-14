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
	"fmt"

	"github.com/cloudstateio/go-support/cloudstate/protocol"
)

func sendFailureAndReturnWith(e error, stream protocol.Crdt_HandleServer) error {
	if err := stream.Send(&protocol.CrdtStreamOut{
		Message: &protocol.CrdtStreamOut_Failure{
			Failure: &protocol.Failure{
				Description: e.Error(),
			},
		},
	}); err != nil {
		return fmt.Errorf("send of CrdtStreamOut#Failure failed: %w", err)
	}
	return e
}
