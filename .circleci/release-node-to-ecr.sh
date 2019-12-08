#!/bin/bash

$(aws ecr get-login --no-include-email --region us-west-2)

DOCKER_TAG=$(./docker/tag.sh)
DOCKER_HASH=$(./docker/hash.sh)

echo "Tagged node Docker image as $NODE_DOCKER_IMAGE:$DOCKER_TAG"
docker tag orbs:export $NODE_DOCKER_IMAGE:$DOCKER_TAG

echo "Tagged node Docker image as $NODE_DOCKER_IMAGE:$DOCKER_HASH"
docker tag orbs:export $NODE_DOCKER_IMAGE:$DOCKER_HASH

docker push $NODE_DOCKER_IMAGE

echo "Tagged signer Docker image as $NODE_DOCKER_IMAGE:$DOCKER_TAG"
docker tag orbs:signer $SIGNER_DOCKER_IMAGE:$DOCKER_TAG

echo "Tagged signer Docker image as $NODE_DOCKER_IMAGE:$DOCKER_HASH"
docker tag orbs:signer $SIGNER_DOCKER_IMAGE:$DOCKER_HASH

docker push $SIGNER_DOCKER_IMAGE