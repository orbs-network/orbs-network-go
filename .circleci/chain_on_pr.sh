#!/bin/bash -e
cd .circleci && npm install @orbs-network/orbs-nebula && cd ..

COMMIT_HASH=$(./docker/hash.sh)

# Determine if we have an active PR
# (The condition means if the variable is not empty)
if [ ! -z "$CI_PULL_REQUESTS" ]
then
    echo "We have an active PR ($CI_PULL_REQUESTS)"

    echo "Downloading the current testnet Boyar config.json"
    curl -O $BOOTSTRAP_URL
    echo "Done downloading!"

    echo "Creating a network for this PR within the config.json file.."
    PR_CHAIN_ID=$(node .circleci/testnet-deploy-new-chain-for-pr.js $CI_PULL_REQUESTS $COMMIT_HASH)
    echo "Done adding a new chain ($PR_CHAIN_ID)"

    echo "Copying the newly updated config.json to S3"
    aws s3 cp --acl public-read config.json $BOOTSTRAP_S3_URI
    echo "Done!"

    echo "Configuration updated, waiting for the new PR chain ($PR_CHAIN_ID) to come up!"

    echo "Sleeping for 2 minutes to allow the network to come up"
    sleep 120
    
    echo "Checking deployment status:"
    node .circleci/check-testnet-deployment.js $PR_CHAIN_ID

    echo "Running the E2E suite against the newly deployed isolated chain for this PR.."
    export API_ENDPOINT=http://$TESTNET_NODE_IP/vchains/$PR_CHAIN_ID/ \
        REMOTE_ENV="true" \
        STRESS_TEST_NUMBER_OF_TRANSACTIONS=100 \
        STRESS_TEST_FAILURE_RATE=20 \
        STRESS_TEST_TARGET_TPS=200 \
        STRESS_TEST=true

    echo "Starting E2E tests against network ($PR_CHAIN_ID)"
    export VCHAIN=$PR_CHAIN_ID
    go test ./test/e2e/... -v

    echo "Disabling the $PR_CHAIN_ID network..."
    rm -f config.json && curl -O $BOOTSTRAP_URL
    node .circleci/testnet-disable-chain.js $PR_CHAIN_ID
    
    echo "Copying the newly updated config.json to S3"
    aws s3 cp --acl public-read config.json $BOOTSTRAP_S3_URI
    echo "Done!"

    echo "Waiting for chain $PR_CHAIN_ID to reflect it's disabled state..."
    sleep 60
    node .circleci/testnet-poll-disabled-chain.js $PR_CHAIN_ID

    echo "E2E tests concluded, Killing the PR network.."
    rm -f config.json && curl -O $BOOTSTRAP_URL
    node .circleci/testnet-remove-chain.js $PR_CHAIN_ID

    echo "Copying the newly updated config.json to S3"
    aws s3 cp --acl public-read config.json $BOOTSTRAP_S3_URI
    echo "Done!"
else
    echo "No active PR, exiting.."
fi

exit 0