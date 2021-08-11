#!/bin/bash

docker login -u $DOCKER_HUB_LOGIN -p $DOCKER_HUB_PASSWORD

DOCKER_HASH=$(./docker/hash.sh)

docker pull orbsnetworkstaging/gamma:$DOCKER_HASH
docker tag orbsnetworkstaging/gamma:$DOCKER_HASH orbs:gamma-server