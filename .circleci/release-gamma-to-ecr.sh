#!/bin/bash

$(aws ecr get-login --no-include-email --region us-west-2)

DOCKER_TAG=$(./docker/tag.sh)
DOCKER_HASH=$(./docker/hash.sh)

echo "Tagged node Docker image as $GAMMA_DOCKER_IMAGE:$DOCKER_TAG"
docker tag orbs:gamma-server $GAMMA_DOCKER_IMAGE:$DOCKER_TAG

echo "Tagged node Docker image as $GAMMA_DOCKER_IMAGE:$DOCKER_HASH"
docker tag orbs:gamma-server $GAMMA_DOCKER_IMAGE:$DOCKER_HASH

docker push $GAMMA_DOCKER_IMAGE
