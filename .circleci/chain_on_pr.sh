#!/bin/bash -e

touch $BASH_ENV
curl -o- https://raw.githubusercontent.com/creationix/nvm/v0.33.11/install.sh | bash
export NVM_DIR="/opt/circleci/.nvm" && . $NVM_DIR/nvm.sh && nvm install v10.14.1
cd .circleci && npm install @orbs-network/orbs-nebula && cd ..

COMMIT_HASH=$(./docker/hash.sh)

# Determine if we have an active PR
if [ ! -z "$CI_PULL_REQUESTS" ]
then
    echo "We have an active PR ($CI_PULL_REQUESTS)"

    echo "Downloading the current testnet Boyar config.json"
    curl -O https://boyar-testnet-bootstrap.s3-us-west-2.amazonaws.com/boyar/config.json
    echo "Done downloading!"

    echo "Creating a network for this PR within the config.json file.."
    PR_CHAIN_ID=$(node .circleci/testnet-deploy-new-chain-for-pr.js $CI_PULL_REQUESTS $COMMIT_HASH)
    echo "Done adding a new chain ($PR_CHAIN_ID)"

    echo "Copying the newly updated config.json to S3"
    aws s3 cp --acl public-read config.json s3://boyar-testnet-bootstrap/boyar/config.json
    echo "Done!"

    echo "Configuration updated, waiting for the new PR chain ($PR_CHAIN_ID) to come up!"

    echo "Sleeping for 10 minutes to allow the network to come up"
    sleep 300
    echo "Still sleeping..."
    sleep 400

    echo "Checking deployment status:"
    node .circleci/check-testnet-deployment.js $PR_CHAIN_ID

    echo "Running the E2E suite against the newly deployed isolated chain for this PR.."
    export API_ENDPOINT=http://35.172.102.63/vchains/$PR_CHAIN_ID/ \
        STRESS_TEST_NUMBER_OF_TRANSACTIONS=100 \
        STRESS_TEST_FAILURE_RATE=20 \
        STRESS_TEST_TARGET_TPS=200 \
        STRESS_TEST=true \

    ./git-submodule-checkout.sh

    go test ./test/e2e/... -v

    # echo "E2E tests concluded, Killing the PR network.."
    # node .circleci/testnet-remove-chain.js $PR_CHAIN_ID

    # echo "Copying the newly updated config.json to S3"
    # aws s3 cp --acl public-read config.json s3://boyar-testnet-bootstrap/boyar/config.json
    # echo "Done!"
else
    echo "No active PR, exiting.."
fi

exit 0