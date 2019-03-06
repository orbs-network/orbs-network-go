#!/bin/bash -xe
export GIT_COMMIT=$(git rev-parse HEAD)
export SEMVER=$(cat ./.version)

LAST_COMMIT_MESSAGE=`git --no-pager log --decorate=short --pretty=oneline -n1 $CIRCLE_SHA1`

BUILD_FLAG=""
if [[ "${LAST_COMMIT_MESSAGE}" == *"#unsafetests"* ]]; then
    BUILD_FLAG="unsafetests"
fi

docker build -f ./docker/build/Dockerfile.build \
    --build-arg GIT_COMMIT=$GIT_COMMIT \
    --build-arg SEMVER=$SEMVER \
    --build-arg BUILD_FLAG=$BUILD_FLAG \
    -t orbs:build .

[ "$(docker ps -a | grep orbs_build)" ] && docker rm -f orbs_build

docker run --name orbs_build orbs:build sleep 1

export SRC=/go/src/github.com/orbs-network/orbs-network-go

rm -rf _bin
docker cp orbs_build:$SRC/_bin .

docker build -f ./docker/build/Dockerfile.export -t orbs:export .
docker build -f ./docker/build/Dockerfile.gamma -t orbs:gamma-server .
