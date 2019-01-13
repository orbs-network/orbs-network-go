#!/bin/bash -xe

rm -rf _logs

export GIT_BRANCH=$(source ./docker/tag.sh)
export GIT_COMMIT=$(git rev-parse HEAD)

rm -rf _tmp/blocks
mkdir -p _tmp/blocks/node{1..4}

cp ./test/e2e/blocks _tmp/blocks/node1
cp ./test/e2e/blocks _tmp/blocks/node2
cp ./test/e2e/blocks _tmp/blocks/node3
cp ./test/e2e/blocks _tmp/blocks/node4

docker-compose -f ./docker/test/docker-compose.yml up --abort-on-container-exit --exit-code-from orbs-e2e