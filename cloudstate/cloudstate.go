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

package cloudstate

import (
	"fmt"
	"net"
	"os"

	"github.com/cloudstateio/go-support/cloudstate/crdt"
	"github.com/cloudstateio/go-support/cloudstate/discovery"
	"github.com/cloudstateio/go-support/cloudstate/eventsourced"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"google.golang.org/grpc"
)

// CloudState is an instance of a Cloudstate User Function
type CloudState struct {
	grpcServer            *grpc.Server
	entityDiscoveryServer *discovery.EntityDiscoveryServer
	eventSourcedServer    *eventsourced.Server
	crdtServer            *crdt.Server
}

// New returns a new CloudState instance.
func New(c protocol.Config) (*CloudState, error) {
	cs := &CloudState{
		grpcServer:            grpc.NewServer(),
		entityDiscoveryServer: discovery.NewServer(c),
		eventSourcedServer:    eventsourced.NewServer(),
		crdtServer:            crdt.NewServer(),
	}
	protocol.RegisterEntityDiscoveryServer(cs.grpcServer, cs.entityDiscoveryServer)
	protocol.RegisterEventSourcedServer(cs.grpcServer, cs.eventSourcedServer)
	protocol.RegisterCrdtServer(cs.grpcServer, cs.crdtServer)
	return cs, nil
}

// RegisterEventSourcedEntity registers an event sourced entity for CloudState.
func (cs *CloudState) RegisterEventSourcedEntity(e *eventsourced.Entity, config protocol.DescriptorConfig) error {
	if err := cs.eventSourcedServer.Register(e); err != nil {
		return err
	}
	if err := cs.entityDiscoveryServer.RegisterEventSourcedEntity(e, config); err != nil {
		return err
	}
	return nil
}

// RegisterCRDT registers a CRDT entity for CloudState.
func (cs *CloudState) RegisterCRDT(e *crdt.Entity, config protocol.DescriptorConfig) error {
	if err := cs.crdtServer.Register(e, e.ServiceName); err != nil {
		return err
	}
	if err := cs.entityDiscoveryServer.RegisterCRDTEntity(e, config); err != nil {
		return err
	}
	return nil
}

// Run runs the CloudState instance.
func (cs *CloudState) Run() error {
	host, ok := os.LookupEnv("HOST")
	if !ok {
		return fmt.Errorf("unable to get environment variable \"HOST\"")
	}
	port, ok := os.LookupEnv("PORT")
	if !ok {
		return fmt.Errorf("unable to get environment variable \"PORT\"")
	}
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	if e := cs.grpcServer.Serve(lis); e != nil {
		return fmt.Errorf("failed to grpcServer.Serve for: %v", lis)
	}
	return nil
}
