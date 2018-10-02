#!/bin/sh -x

export GIT_BRANCH=$(. ./docker-tag.sh)
export GIT_COMMIT=$(git rev-parse HEAD)

# it's way better to store metrics as one file query-wise
cat logs/acceptance/* | sort > logs/acceptance.node.log

find logs -name '*node*.log' -exec bzip2 {} \; -exec aws s3 cp {}.bz2 s3://orbs-network-logs-ci/logs/$(date +year=%Y/month=%m/day=%d)/branch=$GIT_BRANCH/commit=$GIT_COMMIT/ \;
