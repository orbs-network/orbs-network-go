#!/bin/bash -xe

rm -rf _logs

export GIT_BRANCH=$(source ./docker/tag.sh)
export GIT_COMMIT=$(git rev-parse HEAD)
export SRC=/go/src/github.com/orbs-network/orbs-network-go

# run in-process tests (unit tests, component tests, acceptance tests, etc)
[ "$(docker ps -a | grep orbs_test)" ] && docker rm -f orbs_test
docker run --name orbs_test orbs:build bash $SRC/test.sh
TEST_EXIT_CODE=$?

rm -rf _out
mkdir -p _out/fast
docker cp orbs_test:$SRC/results.xml _out/fast
docker cp orbs_test:$SRC/test.out _out/fast
if [[ "$TEST_EXIT_CODE" != 0 ]] ; then exit "$TEST_EXIT_CODE" ; fi

# prepare persistent blocks for docker tests

rm -rf _tmp/blocks
mkdir -p _tmp/blocks/node{1..3}

cp ./test/e2e/blocks _tmp/blocks/node1
cp ./test/e2e/blocks _tmp/blocks/node2
cp ./test/e2e/blocks _tmp/blocks/node3

# run docker-reliant tests
docker-compose -f ./docker/test/docker-compose.yml up --abort-on-container-exit --exit-code-from orbs-e2e
