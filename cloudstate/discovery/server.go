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

// Package discovery implements the Cloudstate entity discovery server.
package discovery

import (
	"sync"

	"context"
	"errors"
	"fmt"
	"log"
	"runtime"

	"github.com/cloudstateio/go-support/cloudstate/crdt"
	"github.com/cloudstateio/go-support/cloudstate/eventsourced"
	"github.com/cloudstateio/go-support/cloudstate/protocol"
	"github.com/golang/protobuf/descriptor"
	"github.com/golang/protobuf/proto"
	filedescr "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/ptypes/empty"
)

const (
	SupportLibraryVersion = "0.2.0"
	SupportLibraryName    = "cloudstate-go-support"
)

// EntityDiscoveryServer implements the Cloudstate discovery protocol.
type EntityDiscoveryServer struct {
	mu                sync.RWMutex
	fileDescriptorSet *filedescr.FileDescriptorSet
	entitySpec        *protocol.EntitySpec
}

// NewServer returns a new and initialized EntityDiscoveryServer.
func NewServer(config protocol.Config) *EntityDiscoveryServer {
	return &EntityDiscoveryServer{
		entitySpec: &protocol.EntitySpec{
			Entities: make([]*protocol.Entity, 0),
			ServiceInfo: &protocol.ServiceInfo{
				ServiceName:           config.ServiceName,
				ServiceVersion:        config.ServiceVersion,
				ServiceRuntime:        fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH),
				SupportLibraryName:    SupportLibraryName,
				SupportLibraryVersion: SupportLibraryVersion,
			},
		},
		fileDescriptorSet: &filedescr.FileDescriptorSet{
			File: make([]*filedescr.FileDescriptorProto, 0),
		},
	}
}

// Discover returns an entity spec for registered entities.
func (s *EntityDiscoveryServer) Discover(_ context.Context, pi *protocol.ProxyInfo) (*protocol.EntitySpec, error) {
	log.Printf("Received discovery call from sidecar [%s %s] supporting Cloudstate %v.%v\n",
		pi.ProxyName,
		pi.ProxyVersion,
		pi.ProtocolMajorVersion,
		pi.ProtocolMinorVersion,
	)
	log.Printf("Responding with: %v\n", s.entitySpec.GetServiceInfo())
	// TODO: s.entitySpec can be written potentially but should not after we started to run the server.
	// check hot to enforce that after CloudState.Run has started.
	return s.entitySpec, nil
}

// ReportError logs any user function error reported by the Cloudstate proxy.
func (s *EntityDiscoveryServer) ReportError(_ context.Context, fe *protocol.UserFunctionError) (*empty.Empty, error) {
	log.Printf("ReportError: %v\n", fe)
	return &empty.Empty{}, nil
}

func (s *EntityDiscoveryServer) updateSpec() (err error) {
	protoBytes, err := proto.Marshal(s.fileDescriptorSet)
	if err != nil {
		return errors.New("unable to Marshal FileDescriptorSet")
	}
	s.entitySpec.Proto = protoBytes
	return nil
}

func (s *EntityDiscoveryServer) resolveFileDescriptors(dc protocol.DescriptorConfig) error {
	if dc.Service != "" {
		if err := s.registerFileDescriptorProto(dc.Service); err != nil {
			return err
		}
	}
	// dependent domain descriptors
	for _, dp := range dc.Domain {
		if err := s.registerFileDescriptorProto(dp); err != nil {
			return err
		}
	}
	for _, dm := range dc.DomainMessages {
		if err := s.registerFileDescriptor(dm); err != nil {
			return err
		}
	}
	return nil
}

func (s *EntityDiscoveryServer) RegisterEventSourcedEntity(e *eventsourced.Entity, config protocol.DescriptorConfig) error {
	if err := s.resolveFileDescriptors(config); err != nil {
		return fmt.Errorf("failed to resolveFileDescriptor for DescriptorConfig: %+v: %w", config, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entitySpec.Entities = append(s.entitySpec.Entities, &protocol.Entity{
		EntityType:    protocol.EventSourced,
		ServiceName:   e.ServiceName.String(),
		PersistenceId: e.PersistenceID,
	})
	return s.updateSpec()
}

func (s *EntityDiscoveryServer) RegisterCRDTEntity(e *crdt.Entity, config protocol.DescriptorConfig) error {
	if err := s.resolveFileDescriptors(config); err != nil {
		return fmt.Errorf("failed to resolveFileDescriptor for DescriptorConfig: %+v: %w", config, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entitySpec.Entities = append(s.entitySpec.Entities, &protocol.Entity{
		EntityType:  protocol.CRDT,
		ServiceName: e.ServiceName.String(),
	})
	return s.updateSpec()
}

func (s *EntityDiscoveryServer) hasRegistered(filename string) bool {
	for _, f := range s.fileDescriptorSet.File {
		if f.GetName() == filename {
			return true
		}
	}
	return false
}

func (s *EntityDiscoveryServer) registerFileDescriptorProto(filename string) error {
	if s.hasRegistered(filename) {
		return nil
	}
	descriptorProto, err := unpackFile(proto.FileDescriptor(filename))
	if err != nil {
		return fmt.Errorf("failed to registerFileDescriptorProto for filename: %s: %w", filename, err)
	}
	s.fileDescriptorSet.File = append(s.fileDescriptorSet.File, descriptorProto)
	for _, fn := range descriptorProto.Dependency {
		err := s.registerFileDescriptorProto(fn)
		if err != nil {
			return err
		}
	}
	return s.updateSpec()
}

func (s *EntityDiscoveryServer) registerFileDescriptor(msg descriptor.Message) error {
	fd, _ := descriptor.ForMessage(msg) // this can panic
	if r := recover(); r != nil {
		return fmt.Errorf("descriptor.ForMessage panicked (%v) for: %+v", r, msg)
	}
	if s.hasRegistered(fd.GetName()) {
		return nil
	}
	s.fileDescriptorSet.File = append(s.fileDescriptorSet.File, fd)
	return nil
}
