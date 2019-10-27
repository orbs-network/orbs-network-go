#!/bin/bash -e

aws --version
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"  # This loads nvm bash_completion

nvm use $NODE_VERSION

export COMMIT_HASH=$(./docker/hash.sh)

cd .circleci && npm install @orbs-network/orbs-nebula && cd ..

curl -O https://boyar-testnet-bootstrap.s3-us-west-2.amazonaws.com/boyar/config.json
node .circleci/testnet-deploy-tag.js $COMMIT_HASH
aws s3 cp --acl public-read config.json s3://boyar-testnet-bootstrap/boyar/config.json

echo "Configuration updated for all nodes in the CI testnet"
echo "Waiting for all nodes to restart and reflect the new version is running"

node .circleci/check-testnet-deployment.js 40
node .circleci/check-testnet-deployment.js 42
node .circleci/check-testnet-deployment.js 2011
node .circleci/check-testnet-deployment.js 2013
