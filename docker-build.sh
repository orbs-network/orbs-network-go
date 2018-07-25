#!/bin/bash -xe

docker build -f Dockerfile.build -t orbs:build .

[ "$(docker ps -a | grep orbs_build)" ] && docker rm -f orbs_build

docker run --name orbs_build orbs:build sleep 1

docker cp orbs_build:/go/src/github.com/orbs-network/orbs-network-go/main .

docker build -f Dockerfile.export -t orbs:export .

