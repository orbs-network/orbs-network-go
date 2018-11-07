#!/bin/bash -xe

docker build -f ./docker/build/Dockerfile.flakiness -t orbs:flakiness .
