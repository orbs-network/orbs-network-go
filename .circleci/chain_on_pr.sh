#!/bin/bash -e

export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"  # This loads nvm bash_completion

nvm use $NODE_VERSION

cd .circleci && npm install @orbs-network/orbs-nebula && cd ..

. ./test.common.sh

COMMIT_HASH=$(./docker/hash.sh)

# Determine if we have an active PR
# (The condition means if the variable is not empty)
if [ ! -z "$CI_PULL_REQUESTS" ]
then
    echo "We have an active PR ($CI_PULL_REQUESTS)"

    echo "Downloading the current testnet Boyar config.json"
    curl -O $BOOTSTRAP_URL
    echo "Done downloading! Let's begin by cleaning up the testnet of any stale networks for PRs which already closed"

    node .circleci/testnet-cleanup-mark.js

    echo "Copying the newly updated config.json to S3 (with networks to remove)"
    aws s3 cp --acl public-read config.json $BOOTSTRAP_S3_URI
    echo "Done!"

    sleep 60
    echo "Verifying the networks are being cleaned.."
    node .circleci/testnet-poll-disabled-chains.js

    echo "Refreshing config.json and removing the dead networks from it.."
    rm -f config.json && curl -O $BOOTSTRAP_URL
    node .circleci/testnet-remove-disabled-chains.js

    echo "Copying the newly updated config.json to S3.."
    aws s3 cp --acl public-read config.json $BOOTSTRAP_S3_URI
    echo "Done!"

    echo "Creating a network for this PR within the config.json file.."
    PR_APP_CHAIN_ID=$(node .circleci/testnet-deploy-new-chain-for-pr.js $CI_PULL_REQUESTS $COMMIT_HASH "APP")
#    PR_MGMT_CHAIN_ID=$(node .circleci/testnet-deploy-new-chain-for-pr.js $CI_PULL_REQUESTS $COMMIT_HASH "MGMT")
    echo "Done adding a new app chain ($PR_APP_CHAIN_ID)"
#    echo "Done adding a new mgmt chain ($PR_MGMT_CHAIN_ID)"

    echo "Copying the newly updated config.json to S3"
    aws s3 cp --acl public-read config.json $BOOTSTRAP_S3_URI
    echo "Done!"

    echo "Configuration updated, waiting for the new PR chain ($PR_APP_CHAIN_ID) to come up!"

    echo "Sleeping for 2 minutes to allow the network to come up"
    sleep 120

    echo "Checking deployment status:"
    node .circleci/check-testnet-deployment.js $PR_APP_CHAIN_ID
#    node .circleci/check-testnet-deployment.js $PR_MGMT_CHAIN_ID

    echo "Running the E2E suite against the newly deployed isolated chain for this PR.."
    export API_ENDPOINT=http://$TESTNET_NODE_IP/vchains/$PR_APP_CHAIN_ID/ \
#        MGMT_API_ENDPOINT=http://$TESTNET_NODE_IP/vchains/$PR_MGMT_CHAIN_ID/ \
        REMOTE_ENV="true" \
        STRESS_TEST_NUMBER_OF_TRANSACTIONS=100 \
        STRESS_TEST_FAILURE_RATE=20 \
        STRESS_TEST_TARGET_TPS=200 \
        STRESS_TEST=true

    echo "Starting E2E tests against networks (app/mgmt): ${PR_APP_CHAIN_ID} / ${PR_MGMT_CHAIN_ID}"
    export VCHAIN=$PR_APP_CHAIN_ID
#    export MGMT_VCHAIN=$PR_MGMT_CHAIN_ID
    time go_test_junit_report e2e_against_$PR_APP_CHAIN_ID -timeout 10m -count=1 ./test/e2e/...

    echo "Disabling the networks (app/mgmt): ${PR_APP_CHAIN_ID} / ${PR_MGMT_CHAIN_ID}"
    rm -f config.json && curl -O $BOOTSTRAP_URL
    node .circleci/testnet-disable-chain.js $PR_APP_CHAIN_ID
#    node .circleci/testnet-disable-chain.js $PR_MGMT_CHAIN_ID

    echo "Copying the newly updated config.json to S3"
    aws s3 cp --acl public-read config.json $BOOTSTRAP_S3_URI
    echo "Done!"

    echo "Waiting for networks (app/mgmt) ${PR_APP_CHAIN_ID} / ${PR_MGMT_CHAIN_ID} to reflect disabled state..."
    sleep 60
    node .circleci/testnet-poll-disabled-chain.js $PR_APP_CHAIN_ID
#    node .circleci/testnet-poll-disabled-chain.js $PR_MGMT_CHAIN_ID

    echo "E2E tests concluded, Killing network.."
    rm -f config.json && curl -O $BOOTSTRAP_URL
    node .circleci/testnet-remove-chain.js $PR_APP_CHAIN_ID
#    node .circleci/testnet-remove-chain.js $PR_MGMT_CHAIN_ID

    echo "Copying the newly updated config.json to S3"
    aws s3 cp --acl public-read config.json $BOOTSTRAP_S3_URI
    echo "Done!"
else
    echo "No active PR, exiting.."
fi

exit 0
