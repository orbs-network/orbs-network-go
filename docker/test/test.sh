#!/bin/bash -xe

rm -rf _logs

export GIT_BRANCH=$(source ./docker/tag.sh)
export GIT_COMMIT=$(git rev-parse HEAD)

export SRC=/go/src/github.com/orbs-network/orbs-network-go

# run in-process tests (unit tests, component tests, acceptance tests, etc)
rm -rf _out
mkdir -p _out
docker-compose -f ./docker/test/docker-compose.tests.yml up --abort-on-container-exit --exit-code-from orbs-tests

# run docker-reliant tests
docker-compose -f ./docker/test/docker-compose.yml up --abort-on-container-exit --exit-code-from orbs-e2e
