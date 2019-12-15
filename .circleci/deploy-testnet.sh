#!/bin/bash -e

# aws --version
# export NVM_DIR="$HOME/.nvm"
# [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm
# [ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"  # This loads nvm bash_completion

# nvm use $NODE_VERSION

# export COMMIT_HASH=$(./docker/hash.sh)

# cd .circleci && npm install @orbs-network/orbs-nebula && cd ..

mkdir -p workspace
echo "$TESTNET_NODE_IP" > workspace/testnet_ip

if [ ! -z "$CI_PULL_REQUESTS" ]
then
    echo "We have an active PR ($CI_PULL_REQUESTS)"
    
    # I use source in this script on purpose so that any exits from the chain on pr scrits
    # will cause this parent script to exit too which is intended by design.
    source ./.circleci/chain_on_pr.sh

    echo "$PR_MGMT_CHAIN_ID" > workspace/mgmt_chain_id
    echo "$PR_APP_CHAIN_ID" > workspace/app_chain_id
else
    echo "Running in master mode"

    curl -O https://boyar-testnet-bootstrap.s3-us-west-2.amazonaws.com/boyar/config.json
    node .circleci/testnet-deploy-tag.js $COMMIT_HASH
    aws s3 cp --acl public-read config.json s3://boyar-testnet-bootstrap/boyar/config.json

    echo "Configuration updated for all nodes in the CI testnet"
    echo "Waiting for all nodes to restart and reflect the new version is running"

    node .circleci/check-testnet-deployment.js 2011
    node .circleci/check-testnet-deployment.js 2013

    echo "2011" > workspace/mgmt_chain_id
    echo "2013" > workspace/app_chain_id
fi

exit 0


