#!/bin/bash -e
rm -f ./_dockerbuild/go.mod.template
cp ./docker/build/go.mod.template ./_dockerbuild/go.mod.template

docker build --no-cache -f ./docker/build/Dockerfile.gamma -t orbs:gamma-server .