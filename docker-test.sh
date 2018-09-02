#!/bin/bash -xe

rm -rf logs
docker-compose up --abort-on-container-exit --exit-code-from orbs-e2e