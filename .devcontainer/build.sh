#!/usr/bin/env bash

set -o nounset
set -o errexit
set -o pipefail

readonly CMD=${1:-image}
readonly IMAGE=cloudstateio/go-support-devcontainer:latest
DOCKER_BUILDKIT=1 docker build . -t "$IMAGE"
if [ "$CMD" == "push" ]; then
  docker push "$IMAGE"
fi
