#!/usr/bin/env bash

set -o nounset
set -o errexit
set -o pipefail

# CloudState protocol

# https://developers.google.com/protocol-buffers/docs/reference/go-generated
# go install google.golang.org/protobuf/cmd/protoc-gen-go
#
# vs
#
# https://grpc.io/docs/quickstart/go/#prerequisites
# go get github.com/golang/protobuf/protoc-gen-go
#
# check this: https://github.com/grpc/grpc-go/pull/3453

protoc --go_out=plugins=grpc,paths=source_relative:cloudstate/protocol --proto_path=protobuf/protocol/cloudstate entity.proto
protoc --go_out=plugins=grpc,paths=source_relative:. --proto_path=protobuf/frontend/ cloudstate/entity_key.proto
protoc --go_out=plugins=grpc,paths=source_relative:cloudstate/entity --proto_path=protobuf/protocol \
  --proto_path=protobuf/protocol/cloudstate crdt.proto
protoc --go_out=plugins=grpc,paths=source_relative:cloudstate/entity --proto_path=protobuf/protocol/ \
  --proto_path=protobuf/protocol/cloudstate function.proto
protoc --go_out=plugins=grpc,paths=source_relative:cloudstate/entity --proto_path=protobuf/protocol/ \
  --proto_path=protobuf/protocol/cloudstate event_sourced.proto

# TCK shopping cart sample
protoc --go_out=plugins=grpc:. \
  --proto_path=protobuf/protocol \
  --proto_path=protobuf/frontend \
  --proto_path=protobuf/frontend/cloudstate \
  --proto_path=protobuf/proxy \
  --proto_path=protobuf/example/shoppingcart shoppingcart.proto
protoc --go_out=plugins=grpc,paths=source_relative:tck/shoppingcart/persistence \
  --proto_path=protobuf/protocol \
  --proto_path=protobuf/frontend \
  --proto_path=protobuf/frontend/cloudstate \
  --proto_path=protobuf/proxy \
  --proto_path=protobuf/example/shoppingcart/persistence domain.proto

# TCK presence sample
protoc --go_out=plugins=grpc:. \
  --proto_path=protobuf/protocol \
  --proto_path=protobuf/frontend \
  --proto_path=protobuf/frontend/cloudstate \
  --proto_path=protobuf/proxy \
  --proto_path=tck/presence/ presence.proto

# TCK CRDT
protoc --go_out=plugins=grpc,paths=source_relative:./tck/proto/crdt \
  --proto_path=protobuf/protocol \
  --proto_path=protobuf/frontend \
  --proto_path=protobuf/frontend/cloudstate \
  --proto_path=protobuf/proxy \
  --proto_path=protobuf/tck tck_crdt.proto
