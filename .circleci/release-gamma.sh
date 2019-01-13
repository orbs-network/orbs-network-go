#!/usr/bin/env bash

if [[ $CIRCLE_TAG == v* ]] ;
then
  GAMMA_VERSION=$CIRCLE_TAG
else
  GAMMA_VERSION=experimental
fi

docker login -u $DOCKER_HUB_LOGIN -p $DOCKER_HUB_PASSWORD

docker tag orbs:gamma-server orbsnetwork/gamma:$GAMMA_VERSION

docker push orbsnetwork/gamma:$GAMMA_VERSION
