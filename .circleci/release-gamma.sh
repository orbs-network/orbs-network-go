#!/usr/bin/env bash

docker login -u $DOCKER_HUB_LOGIN -p $DOCKER_HUB_PASSWORD

docker tag orbs:gamma-server orbsnetwork/gamma:$CIRCLE_TAG
docker tag orbs:gamma-server orbsnetwork/gamma:latest

docker push orbsnetwork/gamma:$CIRCLE_TAG
docker push orbsnetwork/gamma:latest