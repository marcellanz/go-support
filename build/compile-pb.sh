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

# TCK chat example
protoc --go_out=plugins=grpc,paths=source_relative:./example/chat/presence/ \
  --proto_path=protobuf/protocol \
  --proto_path=protobuf/frontend \
  --proto_path=protobuf/frontend/cloudstate \
  --proto_path=protobuf/proxy \
  --proto_path=example/chat/presence/ example/chat/presence/presence.proto

  protoc --go_out=plugins=grpc,paths=source_relative:./example/chat/friends/ \
  --proto_path=protobuf/protocol \
  --proto_path=protobuf/frontend \
  --proto_path=protobuf/frontend/cloudstate \
  --proto_path=protobuf/proxy \
  --proto_path=example/chat/friends/ example/chat/friends/friends.proto

# TCK CRDT
protoc --go_out=plugins=grpc,paths=source_relative:./tck/proto/crdt \
  --proto_path=protobuf/protocol \
  --proto_path=protobuf/frontend \
  --proto_path=protobuf/frontend/cloudstate \
  --proto_path=protobuf/proxy \
  --proto_path=protobuf/tck tck_crdt.proto

# TCK Eventsourced
protoc --go_out=plugins=grpc,paths=source_relative:./tck/pb/eventsourced \
  --proto_path=protobuf/protocol \
  --proto_path=protobuf/frontend \
  --proto_path=protobuf/frontend/cloudstate \
  --proto_path=protobuf/proxy \
  --proto_path=protobuf/tck/cloudstate/tck/model \
  --proto_path=protobuf/tck eventsourced.proto
