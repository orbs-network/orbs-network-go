#!/bin/bash -xe

docker build -f ./build/docker/Dockerfile.flakiness -t orbs:flakiness .
