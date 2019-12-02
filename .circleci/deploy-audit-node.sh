#!/bin/bash -e

aws --version
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"  # This loads nvm bash_completion

nvm use $NODE_VERSION

if [[ $CIRCLE_TAG == v* ]] ;
then
  VERSION=$CIRCLE_TAG
else
  VERSION=$(./docker/hash.sh)
fi

cd .circleci && npm install @orbs-network/orbs-nebula && cd ..

echo "Downloading current Orbs Core Audit Node configuration from S3.."
curl -O https://orbs-core-audit-node.s3.us-east-2.amazonaws.com/config.json

echo "Updating all vchains in the audit node to point to the newly released version (orbsnetwork/node:$VERSION)"
node .circleci/mainnet-deploy-tag.js $VERSION
aws s3 cp --acl public-read config.json s3://orbs-core-audit-node/config.json
echo "Configuration uploaded to S3!"

echo "Checking deployment status on the Audit node.."
node .circleci/check-audit-node-deployment.js 1960000
