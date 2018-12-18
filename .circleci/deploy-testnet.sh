#!/bin/bash -e

touch $BASH_ENV
curl -o- https://raw.githubusercontent.com/creationix/nvm/v0.33.11/install.sh | bash

export NVM_DIR="/opt/circleci/.nvm" && . $NVM_DIR/nvm.sh && nvm install v10.14.1 && nvm use v10.14.1

curl https://releases.hashicorp.com/terraform/0.11.10/terraform_0.11.10_linux_amd64.zip -o tf.zip && unzip tf.zip && sudo mv terraform /usr/bin/

echo $TESTNET_SSH_PUBLIC_KEY > ~/.ssh/id_rsa.pub

git clone https://github.com/orbs-network/nebula && cd nebula && git checkout testnet
npm install

export DOCKER_TAG_SHA256=$(docker images $DOCKER_IMAGE --digests --format '{{.Digest}} {{.Tag}}' | grep master | cut -d ' ' -f 1)

#aws s3 sync s3://orbs-network-config-staging/nebula/cache/_terraform _terraform

export REGIONS=us-east-1,eu-central-1,ap-northeast-1,ap-northeast-2,ap-southeast-2,ca-central-1
node deploy.js --regions $REGIONS --update-vchains --chain-version master@$DOCKER_TAG_SHA256

#rm -rf _terraform/*/.terraform && aws s3 sync _terraform s3://orbs-network-config-staging/nebula/cache/_terraform