#!/bin/bash

docker login -u $DOCKER_HUB_LOGIN -p $DOCKER_HUB_PASSWORD

./docker/build/create-version-file.sh
export VERSION=$(cat .version)

docker tag orbs:export orbsnetworkstaging/node:$VERSION
docker push orbsnetworkstaging/node:$VERSION
