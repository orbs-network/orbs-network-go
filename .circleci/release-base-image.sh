#!/bin/bash -e

$(aws ecr get-login --no-include-email --region us-west-2)
docker login -u $DOCKER_HUB_LOGIN -p $DOCKER_HUB_PASSWORD

docker build -f ./docker/build/Dockerfile.base -t orbsnetwork/base:latest
docker push orbsnetwork/base:latest
