#!/bin/bash -e

. ./docker/build/create-version-file.sh # will exit with code=2 if tag has invalid format, which will terminate this script because of the -e shebang option

BUILD_CMD="./build-node.sh" ./docker/build/build-binaries-in-superfluous-container.sh

docker build -f ./docker/build/Dockerfile.export -t orbs:export .
docker build -f ./docker/build/Dockerfile.signer -t orbs:signer .

if [[ $ORBS_EXPERIMENTAL == "true" ]] ;
then
  docker build -f ./docker/build/Dockerfile.export.experimental -t orbs:export .
fi
