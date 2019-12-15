#!/bin/bash -e

echo "Disabling the networks (app/mgmt): ${PR_APP_CHAIN_ID} / ${PR_MGMT_CHAIN_ID}"
rm -f config.json && curl -O $BOOTSTRAP_URL
node .circleci/testnet-disable-chain.js $PR_APP_CHAIN_ID
node .circleci/testnet-disable-chain.js $PR_MGMT_CHAIN_ID

echo "Copying the newly updated config.json to S3"
aws s3 cp --acl public-read config.json $BOOTSTRAP_S3_URI
echo "Done!"

echo "Waiting for networks (app/mgmt) ${PR_APP_CHAIN_ID} / ${PR_MGMT_CHAIN_ID} to reflect disabled state..."
sleep 60
node .circleci/testnet-poll-disabled-chain.js $PR_APP_CHAIN_ID
node .circleci/testnet-poll-disabled-chain.js $PR_MGMT_CHAIN_ID

echo "E2E tests concluded, Killing network.."
rm -f config.json && curl -O $BOOTSTRAP_URL
node .circleci/testnet-remove-chain.js $PR_APP_CHAIN_ID
node .circleci/testnet-remove-chain.js $PR_MGMT_CHAIN_ID

echo "Copying the newly updated config.json to S3"
aws s3 cp --acl public-read config.json $BOOTSTRAP_S3_URI
echo "Done!"