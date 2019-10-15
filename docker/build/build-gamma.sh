#!/bin/bash -e

docker build --no-cache -f ./docker/build/Dockerfile.gamma -t orbs:gamma-server .