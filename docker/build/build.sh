#!/bin/bash -ex
export GIT_COMMIT=$(git rev-parse HEAD)
export SEMVER=$(cat ./.version)

LAST_COMMIT_MESSAGE=`git --no-pager log --decorate=short --pretty=oneline -n1 $CIRCLE_SHA1`

BUILD_FLAG=""
if [[ "${LAST_COMMIT_MESSAGE}" == *"#unsafetests"* ]]; then
    BUILD_FLAG="unsafetests"
fi

docker build --no-cache -f ./docker/build/Dockerfile.build \
    --build-arg GIT_COMMIT=$GIT_COMMIT \
    --build-arg SEMVER=$SEMVER \
    --build-arg BUILD_FLAG=$BUILD_FLAG \
    -t orbs:build .

[ "$(docker ps -a | grep orbs_build)" ] && docker rm -f orbs_build

docker run --name orbs_build orbs:build sleep 1

export SRC=/src

rm -rf _bin && mkdir -p _bin _dockerbuild
rm -f ./_dockerbuild/go.mod.template
SDK_VERSION=$(cat go.mod | grep orbs-contract-sdk | awk '{print $2}')
cp ./docker/build/go.mod.template ./_dockerbuild/go.mod.t
sed "s/SDK_VER/$SDK_VERSION/g" _dockerbuild/go.mod.t > _dockerbuild/go.mod.template

docker cp orbs_build:$SRC/_bin .

docker build -f ./docker/build/Dockerfile.export -t orbs:export .
docker build -f ./docker/build/Dockerfile.signer -t orbs:signer .
./docker/build/build-gamma.sh

# Builds experimental features (extra libraries)
if [[ $CIRCLE_TAG != v* ]] ;
then
    docker build -f ./docker/build/Dockerfile.export.experimental -t orbs:export .
    docker build -f ./docker/build/Dockerfile.gamma.experimental -t orbs:gamma-server .
fi