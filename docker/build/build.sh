#!/bin/bash -xe
export SKIP_TESTS=${SKIP_TESTS-false}
export GIT_COMMIT=$(git rev-parse HEAD)
export SEMVER=$(cat ./.version)

docker build -f ./docker/build/Dockerfile.build \
    --build-arg SKIP_TESTS=$SKIP_TESTS \
    --build-arg GIT_COMMIT=$GIT_COMMIT \
    --build-arg SEMVER=$SEMVER \
    -t orbs:build .

[ "$(docker ps -a | grep orbs_build)" ] && docker rm -f orbs_build

docker run --name orbs_build orbs:build sleep 1

export SRC=/go/src/github.com/orbs-network/orbs-network-go

rm -rf _bin
docker cp orbs_build:$SRC/_bin .

docker build -f ./docker/build/Dockerfile.export -t orbs:export .
docker build -f ./docker/build/Dockerfile.gamma -t orbs:gamma-server .

docker build -f ./docker/build/Dockerfile.e2e -t orbs:e2e .
docker build -f ./docker/build/Dockerfile.external -t orbs:external .
