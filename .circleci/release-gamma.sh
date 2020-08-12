#!/bin/bash -e

if [[ $CIRCLE_TAG == v* ]] ;
then
  VERSION=$CIRCLE_TAG
else
  VERSION=experimental
fi

$(aws ecr get-login --no-include-email --region us-west-2)
docker login -u $DOCKER_HUB_LOGIN -p $DOCKER_HUB_PASSWORD

# we are only releasing gamma
# node releases are manual

docker pull $GAMMA_DOCKER_IMAGE:$(./docker/hash.sh)

docker tag $GAMMA_DOCKER_IMAGE:$(./docker/hash.sh) orbsnetwork/gamma:$VERSION
docker push orbsnetwork/gamma:$VERSION
