#!/bin/bash -xe

GO_MOD_TEMPLATE="./docker/build/go.mod.template"

export GIT_COMMIT=$(git rev-parse HEAD)
export SEMVER=$(cat ./.version)

if [ -z "$BUILD_FLAG" ]; then
    LAST_COMMIT_MESSAGE=`git --no-pager log --decorate=short --pretty=oneline -n1 $CIRCLE_SHA1`

    if [[ "${LAST_COMMIT_MESSAGE}" == *"#unsafetests"* ]]; then
        BUILD_FLAG="unsafetests"
    fi
fi

docker build --no-cache -f ./docker/build/Dockerfile.build \
    --build-arg GIT_COMMIT=$GIT_COMMIT \
    --build-arg SEMVER=$SEMVER \
    --build-arg BUILD_FLAG=$BUILD_FLAG \
    --build-arg BUILD_CMD=$BUILD_CMD \
    -t orbs:build .

[ "$(docker ps -a | grep orbs_build)" ] && docker rm -f orbs_build

docker run --name orbs_build orbs:build sleep 1

export SRC=/src

docker cp orbs_build:$SRC/_bin .

