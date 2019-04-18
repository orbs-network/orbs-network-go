#!/bin/bash

$(aws ecr get-login --no-include-email --region us-west-2)

docker tag orbs:export $NODE_DOCKER_IMAGE:$(./docker/tag.sh)
docker tag orbs:export $NODE_DOCKER_IMAGE:$(./docker/hash.sh)

docker push $NODE_DOCKER_IMAGE

docker tag orbs:export $GAMMA_DOCKER_IMAGE:$(./docker/tag.sh)
docker tag orbs:export $GAMMA_DOCKER_IMAGE:$(./docker/hash.sh)

docker push $GAMMA_DOCKER_IMAGE