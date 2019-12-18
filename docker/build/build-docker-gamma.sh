#!/bin/bash -e

. ./docker/build/create-version-file.sh # will exit with code=2 if tag has invalid format, which will terminate this script because of the -e shebang option

BUILD_CMD="./build-gamma.sh" ./docker/build/build-binaries-in-superfluous-container.sh

docker build --no-cache -f ./docker/build/Dockerfile.gamma -t orbs:gamma-server .