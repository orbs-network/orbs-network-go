#!/bin/bash -xe

rm -rf _logs

export GIT_BRANCH=$(source ./docker/tag.sh)
export GIT_COMMIT=$(git rev-parse HEAD)

export SRC=/go/src/github.com/orbs-network/orbs-network-go

docker run orbs:build bash $SRC/test.sh

mkdir -p /tmp/test-results/
docker cp orbs_build:$SRC/report.xml /tmp/test-results/

docker-compose -f ./docker/test/docker-compose.yml up --abort-on-container-exit --exit-code-from orbs-e2e
