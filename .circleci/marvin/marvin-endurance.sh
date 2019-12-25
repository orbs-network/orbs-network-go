#!/bin/bash -e

APP_CHAIN_ID=$(cat workspace/app_chain_id)
TESTNET_IP=$(cat workspace/testnet_ip)

if [[ -z "${APP_CHAIN_ID}" ]]; then
  echo "Environment variable APP_CHAIN_ID is not set"
  exit 1
fi

if [[ -z "${TESTNET_IP}" ]]; then
  echo "Environment variable TESTNET_IP is not set"
  exit 1
fi

aws --version

# If running locally, need to disable these next 4 lines
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"                   # This loads nvm
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion" # This loads nvm bash_completion
nvm use "${NODE_VERSION}"

cd .circleci && npm install && cd ..

# Get the vchains to act upon from CircleCI's workspace
echo "Running Marvin tests on deployed app chain ($APP_CHAIN_ID) on IP: $TESTNET_IP"

./.circleci/marvin/marvin-endurance.js "${APP_CHAIN_ID}" "${TESTNET_IP}" workspace/job_id
