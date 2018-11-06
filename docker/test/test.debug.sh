#!/bin/bash -xe

rm -rf _logs
docker-compose -f ./docker/test/docker-compose.debug.yml up --abort-on-container-exit --exit-code-from orbs-e2e