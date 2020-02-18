#!/bin/bash -e
VERSION=experimental
VHASH=$(./docker/hash.sh)

$(aws ecr get-login --no-include-email --region us-west-2)
docker login -u $DOCKER_HUB_LOGIN -p $DOCKER_HUB_PASSWORD

docker pull $NODE_DOCKER_IMAGE:$VHASH

docker tag $NODE_DOCKER_IMAGE:$VHASH orbsnetwork/node:$VERSION
docker push orbsnetwork/node:$VERSION

docker tag $NODE_DOCKER_IMAGE:$VHASH orbsnetwork/node:$VHASH
docker push orbsnetwork/node:$VHASH

docker pull $GAMMA_DOCKER_IMAGE:$VHASH
docker tag $GAMMA_DOCKER_IMAGE:$VHASH orbsnetwork/gamma:$VERSION
docker push orbsnetwork/gamma:$VERSION

docker pull $SIGNER_DOCKER_IMAGE:$VHASH
docker tag $SIGNER_DOCKER_IMAGE:$VHASH orbsnetwork/signer:$VERSION
docker push orbsnetwork/signer:$VERSION
