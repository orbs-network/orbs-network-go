#!/bin/bash -e

GO_MOD_TEMPLATE=./docker/build/go.mod.template
if [[ $CIRCLE_TAG != v* ]] ;
then
  export ORBS_EXPERIMENTAL="true"
  GO_MOD_TEMPLATE=./docker/build/go.mod.template.experimental
fi

if [[ ! -z "$CIRCLE_TAG" ]]; then
    echo "This is a release run - Updating the .version file to indicate the correct Semver"
    echo "For this release ($CIRCLE_TAG)..."

    TAG_FIRST_CHAR=$(echo "$CIRCLE_TAG" | head -c 1)
    if [[ $TAG_FIRST_CHAR != "v" ]]; then
        echo "Oops! the tag format supplied is invalid while releasing a new version of the Orbs node"
        echo "Tag supplied is $CIRCLE_TAG and we do not allow that. Must use format vX.X.X!"
        exit 2
    fi

    echo "$CIRCLE_TAG" > .version
else
    LATEST_SEMVER=$(git describe --tags --abbrev=0)
    SHORT_COMMIT=$(git rev-parse HEAD | cut -c1-8)
    echo "$LATEST_SEMVER-$SHORT_COMMIT" > .version
fi

export GIT_COMMIT=$(git rev-parse HEAD)
export SEMVER=$(cat ./.version)

if [ -z "$BUILD_FLAG" ]; then
    LAST_COMMIT_MESSAGE=`git --no-pager log --decorate=short --pretty=oneline -n1 $CIRCLE_SHA1`

    if [[ "${LAST_COMMIT_MESSAGE}" == *"#unsafetests"* ]]; then
        BUILD_FLAG="unsafetests"
    fi
fi

BUILD_CMD="./build-node.sh"

docker build --no-cache -f ./docker/build/Dockerfile.build \
    --build-arg GIT_COMMIT=$GIT_COMMIT \
    --build-arg SEMVER=$SEMVER \
    --build-arg BUILD_FLAG=$BUILD_FLAG \
    --build-arg BUILD_CMD=$BUILD_CMD \
    -t orbs:build .

[ "$(docker ps -a | grep orbs_build)" ] && docker rm -f orbs_build

docker run --name orbs_build orbs:build sleep 1

export SRC=/src

rm -rf _bin && mkdir -p _bin _dockerbuild
rm -f ./_dockerbuild/go.mod.template
SDK_VERSION=$(cat go.mod | grep orbs-contract-sdk | awk '{print $2}')
cp $GO_MOD_TEMPLATE ./_dockerbuild/go.mod.t
sed "s/SDK_VER/$SDK_VERSION/g" _dockerbuild/go.mod.t > _dockerbuild/go.mod.template

docker cp orbs_build:$SRC/_bin .

docker build -f ./docker/build/Dockerfile.export -t orbs:export .
docker build -f ./docker/build/Dockerfile.signer -t orbs:signer .

if [[ $ORBS_EXPERIMENTAL == "true" ]] ;
then
  docker build -f ./docker/build/Dockerfile.export.experimental -t orbs:export .
fi