#!/bin/bash -xe

export CONSENSUSALGO=${CONSENSUSALGO-benchmark}

rm -rf _logs

export GIT_BRANCH=$(source ./docker/tag.sh)
export GIT_COMMIT=$(git rev-parse HEAD)
export SRC=/go/src/github.com/orbs-network/orbs-network-go

# prepare persistent blocks for docker tests

rm -rf _tmp/blocks
mkdir -p _tmp/blocks/node{1..4}

cp ./test/e2e/_data/blocks _tmp/blocks/node1
cp ./test/e2e/_data/blocks _tmp/blocks/node2
cp ./test/e2e/_data/blocks _tmp/blocks/node3
cp ./test/e2e/_data/blocks _tmp/blocks/node4

# run docker-reliant tests
export GANACHE_START_TIME=$(node -e "console.log(new Date(new Date() - 1000 * 60 * 25))")
docker-compose -f ./docker/test/docker-compose.yml up --abort-on-container-exit --exit-code-from orbs-e2e
