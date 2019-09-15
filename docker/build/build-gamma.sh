#!/bin/bash -e
rm -f ./_bin/go.mod.template
cp ./docker/build/go.mod.template ./_bin/go.mod.template

docker build --no-cache -f ./docker/build/Dockerfile.gamma -t orbs:gamma-server .