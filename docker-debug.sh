#!/bin/bash -xe

rm -rf logs
docker-compose -f docker-compose.debug.yml up --build --abort-on-container-exit --exit-code-from orbs-e2e