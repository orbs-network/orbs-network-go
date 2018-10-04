#!/bin/bash -xe

docker build -f Dockerfile.build -t orbs:build .

[ "$(docker ps -a | grep orbs_build)" ] && docker rm -f orbs_build

docker run --name orbs_build orbs:build sleep 1

export SRC=/go/src/github.com/orbs-network/orbs-network-go

docker cp orbs_build:$SRC/orbs-node .
docker cp orbs_build:$SRC/gamma-cli .
docker cp orbs_build:$SRC/gamma-server .

docker cp orbs_build:$SRC/e2e.test .

docker build -f Dockerfile.export -t orbs:export .
docker build -f Dockerfile.gamma -t orbs:gamma-server .

docker build -f Dockerfile.e2e -t orbs:e2e .