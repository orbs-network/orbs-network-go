#!/bin/bash -xe

rm -rf logs

export GIT_BRANCH=$(source ./docker-tag.sh)
export GIT_COMMIT=$(git rev-parse HEAD)

docker-compose up --abort-on-container-exit --exit-code-from orbs-e2e