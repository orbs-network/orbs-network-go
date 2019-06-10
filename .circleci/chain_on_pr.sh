#!/bin/bash -xe

COMMIT_HASH=$(./docker/hash.sh)

# Determine if we have an active PR
if [ ! -z "$CI_PULL_REQUESTS" ]
then
    echo "We have an active PR ($CI_PULL_REQUESTS)"
    curl -O https://s3.eu-central-1.amazonaws.com/boyar-ci/boyar/config.json
    PR_CHAIN_ID=$(node .circleci/testnet-deploy-new-chain-for-pr.js $CI_PULL_REQUESTS $COMMIT_HASH)

    aws s3 cp --acl public-read config.json s3://boyar-ci/boyar/config.json

    echo "Configuration updated, waiting for the new PR chain ($PR_CHAIN_ID) to come up!"

    sleep 20

    node .circleci/check-testnet-deployment.js

    export API_ENDPOINT=http://35.172.102.63/vchains/$PR_CHAIN_ID/ \
        STRESS_TEST_NUMBER_OF_TRANSACTIONS=100 \
        STRESS_TEST_FAILURE_RATE=20 \
        STRESS_TEST_TARGET_TPS=200 \
        STRESS_TEST=true \

    go test ./test/e2e/... -v
else
    echo "No active PR, exiting.."
fi

exit 0