#!/bin/bash -e

touch $BASH_ENV
curl -o- https://raw.githubusercontent.com/creationix/nvm/v0.33.11/install.sh | bash

export NVM_DIR="/opt/circleci/.nvm" && . $NVM_DIR/nvm.sh && nvm install v10.14.1 && nvm use v10.14.1

echo $TESTNET_SSH_PUBLIC_KEY > ~/.ssh/id_rsa.pub

export COMMIT_HASH=$(./docker/hash.sh)

git clone https://github.com/orbs-network/nebula && cd nebula && git checkout testnet
npm install

export REGIONS=us-east-1,eu-central-1,ap-northeast-1,ap-northeast-2,sa-east-1,ca-central-1
node deploy.js --regions $REGIONS --update-vchains --chain-version $COMMIT_HASH
