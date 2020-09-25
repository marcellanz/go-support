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

protoc --go_out=plugins=grpc,paths=source_relative:. --proto_path=protobuf/frontend/ protobuf/frontend/cloudstate/entity_key.proto
protoc --go_out=plugins=grpc:. --proto_path=protobuf/protocol/ protobuf/protocol/cloudstate/entity.proto
protoc --go_out=plugins=grpc:. --proto_path=protobuf/protocol/ protobuf/protocol/cloudstate/event_sourced.proto
protoc --go_out=plugins=grpc:. --proto_path=protobuf/protocol/ protobuf/protocol/cloudstate/function.proto
protoc --go_out=plugins=grpc:. --proto_path=protobuf/protocol/ protobuf/protocol/cloudstate/crdt.proto

# TCK shopping cart sample
protoc --go_out=plugins=grpc:. \
  --proto_path=protobuf/protocol \
  --proto_path=protobuf/frontend \
  --proto_path=protobuf/proxy \
  --proto_path=protobuf/example protobuf/example/shoppingcart/shoppingcart.proto
protoc --go_out=plugins=grpc,paths=source_relative:tck/shoppingcart/persistence \
  --proto_path=protobuf/protocol \
  --proto_path=protobuf/frontend/ \
  --proto_path=protobuf/proxy \
  --proto_path=protobuf/example/shoppingcart/persistence protobuf/example/shoppingcart/persistence/domain.proto

# TCK presence sample
protoc --go_out=plugins=grpc:. \
  --proto_path=protobuf/protocol/ \
  --proto_path=protobuf/frontend/ \
  --proto_path=protobuf/proxy/ \
  --proto_path=tck/presence/ \
  tck/presence/presence.proto

# TCK
protoc --go_out=plugins=grpc,paths=source_relative:./tck/proto/crdt \
  --proto_path=protobuf/protocol \
  --proto_path=protobuf/frontend \
  --proto_path=protobuf/proxy \
  --proto_path=protobuf/tck \
  protobuf/tck/tck_crdt.proto
