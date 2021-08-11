#!/bin/bash -e

docker login -u $DOCKER_HUB_LOGIN -p $DOCKER_HUB_PASSWORD

DOCKER_TAG=$(./docker/tag.sh)
DOCKER_HASH=$(./docker/hash.sh)

echo "Tagged node Docker image as orbsnetworkstaging/gamma:$DOCKER_TAG"
docker tag orbs:gamma-server orbsnetworkstaging/gamma:$DOCKER_TAG

echo "Tagged node Docker image as orbsnetworkstaging/gamma:$DOCKER_HASH"
docker tag orbs:gamma-server orbsnetworkstaging/gamma:$DOCKER_HASH

docker push orbsnetworkstaging/gamma
