#!/bin/bash -xe

rm -rf _logs

export GIT_BRANCH=$(source ./docker/tag.sh)
export GIT_COMMIT=$(git rev-parse HEAD)
export SRC=/go/src/github.com/orbs-network/orbs-network-go

# run in-process tests (unit tests, component tests, acceptance tests, etc)
rm -rf _out
mkdir -p _out
docker-compose -f ./docker/test/docker-compose.tests.yml up --abort-on-container-exit --exit-code-from orbs-tests

# prepare persistent blocks for docker tests

rm -rf _tmp/blocks
mkdir -p _tmp/blocks/node{1..4}

cp ./test/e2e/_data/blocks _tmp/blocks/node1
cp ./test/e2e/_data/blocks _tmp/blocks/node2
cp ./test/e2e/_data/blocks _tmp/blocks/node3
cp ./test/e2e/_data/blocks _tmp/blocks/node4

# run docker-reliant tests
docker-compose -f ./docker/test/docker-compose.yml up --abort-on-container-exit --exit-code-from orbs-e2e
