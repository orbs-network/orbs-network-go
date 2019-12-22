#!/bin/bash -e

aws --version
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"  # This loads nvm bash_completion

nvm use "${NODE_VERSION}"

cd .circleci && npm install && cd ..

# Get the vchains to act upon from CircleCI's workspace
echo "Running Marvin tests on deployed app chain ($APP_CHAIN_ID) on IP: $TESTNET_IP"

./.circleci/marvin-endurance.js "${APP_CHAIN_ID}" "${TESTNET_IP}" workspace/job_id
