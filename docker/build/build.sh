#!/bin/bash -e

if [[ $CIRCLE_TAG != v* ]] ;
then
  export ORBS_EXPERIMENTAL="true"
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

GO_MOD_TEMPLATE=./docker/build/go.mod.template
DOCKERFILE_SIGNER=./docker/build/Dockerfile.signer
DOCKERFILE_EXPORT=./docker/build/Dockerfile.export
DOCKERFILE_GAMMA=./docker/build/Dockerfile.gamma

if [[ $ORBS_EXPERIMENTAL == "true" ]] ;
then
  GO_MOD_TEMPLATE=./docker/build/go.mod.template.experimental
  DOCKERFILE_EXPORT=./docker/build/Dockerfile.export.experimental
  DOCKERFILE_GAMMA=./docker/build/Dockerfile.gamma.experimental
fi

rm -rf _bin && mkdir -p _bin _dockerbuild
rm -f ./_dockerbuild/go.mod.template
SDK_VERSION=$(cat go.mod | grep orbs-contract-sdk | awk '{print $2}')
cp $GO_MOD_TEMPLATE ./_dockerbuild/go.mod.t
sed "s/SDK_VER/$SDK_VERSION/g" _dockerbuild/go.mod.t > _dockerbuild/go.mod.template

docker cp orbs_build:$SRC/_bin .

docker build -f $DOCKERFILE_EXPORT -t orbs:export .
docker build -f $DOCKERFILE_SIGNER -t orbs:signer .
docker build --no-cache -f $DOCKERFILE_GAMMA -t orbs:gamma-server .