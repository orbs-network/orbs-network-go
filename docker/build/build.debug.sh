#!/bin/bash -xe

docker build -f ./docker/build/Dockerfile.build -t orbs:debug --build-arg SKIP_DEVTOOLS=true --build-arg .
