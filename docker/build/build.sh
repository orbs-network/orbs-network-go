#!/bin/bash -xe

docker build -f ./docker/build/Dockerfile.build -t orbs:build .

[ "$(docker ps -a | grep orbs_build)" ] && docker rm -f orbs_build

docker run --name orbs_build orbs:build sleep 1

export SRC=/go/src/github.com/orbs-network/orbs-network-go

rm -rf _bin
docker cp orbs_build:$SRC/_bin .

docker build -f ./docker/build/Dockerfile.export -t orbs:export .
docker build -f ./docker/build/Dockerfile.gamma -t orbs:gamma-server .

docker build -f ./docker/build/Dockerfile.e2e -t orbs:e2e .