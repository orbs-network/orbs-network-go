#!/bin/bash -e

aws --version
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"  # This loads nvm bash_completion

nvm use $NODE_VERSION

cd .circleci && npm install && cd ..

# Get the vchains to act upon from CircleCI's workspace
VCHAIN=$(cat workspace/app_chain_id)
TESTNET_IP=$(cat workspace/testnet_ip)

echo "Running Marvin tests on deployed app chain ($VCHAIN)"
echo "on IP: $TESTNET_IP"

./.circleci/marvin-endurance.js $VCHAIN $TESTNET_IP workspace/job_id
