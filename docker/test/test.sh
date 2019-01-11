#!/bin/bash -xe

rm -rf _logs

export GIT_BRANCH=$(source ./docker/tag.sh)
export GIT_COMMIT=$(git rev-parse HEAD)

export SRC=/go/src/github.com/orbs-network/orbs-network-go

# run in-process tests (unit tests, component tests, acceptance tests, etc)
docker run --name orbs_test orbs:build bash $SRC/test.sh
mkdir -p /tmp/test-results/
docker cp orbs_test:$SRC/report.xml /tmp/test-results/

# run docker-reliant tests
docker-compose -f ./docker/test/docker-compose.yml up --abort-on-container-exit --exit-code-from orbs-e2e
