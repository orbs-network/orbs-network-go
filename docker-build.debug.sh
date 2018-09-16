#!/bin/bash -xe

docker build -f Dockerfile.build -t orbs:debug --build-arg SKIP_DEVTOOLS=true --build-arg SKIP_TESTS=true .
