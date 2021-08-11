#!/bin/bash

docker login -u $DOCKER_HUB_LOGIN -p $DOCKER_HUB_PASSWORD

./docker/build/create-version-file.sh
export VERSION=$(cat .version)

NODE_DOCKER_IMAGE="orbsnetworkstaging/node:$VERSION"
docker pull $NODE_DOCKER_IMAGE
docker tag $NODE_DOCKER_IMAGE orbs:export
