#!/bin/bash

DOCKER_HASH=$(./docker/hash.sh)
ECR_IMAGE="727534866935.dkr.ecr.us-west-2.amazonaws.com/orbs-network-v1:$DOCKER_HASH"
docker pull $ECR_IMAGE
docker tag $ECR_IMAGE orbs:export

ECR_IMAGE="727534866935.dkr.ecr.us-west-2.amazonaws.com/orbs-network-signer:$DOCKER_HASH"
docker pull $ECR_IMAGE
docker tag $ECR_IMAGE orbs:signer