#!/bin/bash

$(aws ecr get-login --no-include-email --region us-west-2)
DOCKER_HASH=$(./docker/hash.sh)
ECR_IMAGE="727534866935.dkr.ecr.us-west-2.amazonaws.com/orbs-network-gamma:$DOCKER_HASH"
docker pull $ECR_IMAGE
docker tag $ECR_IMAGE orbs:gamma-server