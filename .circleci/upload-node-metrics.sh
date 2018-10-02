#!/usr/bin/env bash

export GIT_BRANCH=$(source ./docker-tag.sh)
export GIT_COMMIT=$(git rev-parse HEAD)

find logs -name 'node*.log' -exec aws s3 cp {} s3://orbs-network-logs-ci/logs/$(date +year=%Y/month=%m/day=%d/)/branch=$GIT_BRANCH/commit=$GIT_COMMIT/ \;
