#!/bin/bash

. ./test.common.sh

# If running locally, need to disable these next 4 lines
if [[ "$CI" == "true" ]]; then
  export NVM_DIR="$HOME/.nvm"
  [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"                   # This loads nvm
  [ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion" # This loads nvm bash_completion
  nvm use "${NODE_VERSION}"
fi

npm install

# Get the vchains to act upon from CircleCI's workspace
VCHAIN=$(cat workspace/app_chain_id)
TESTNET_IP=$(cat workspace/testnet_ip)

echo "Downloading the current testnet Boyar config.json"
curl -O $BOOTSTRAP_URL

node .circleci/testnet/check-deployment.js $VCHAIN

echo "Running E2E on deployed app chain ($VCHAIN)"
echo "on IP: $TESTNET_IP"

export API_ENDPOINT=http://$TESTNET_IP/vchains/$VCHAIN/ \
    REMOTE_ENV="true" \
    STRESS_TEST_NUMBER_OF_TRANSACTIONS=100 \
    STRESS_TEST_FAILURE_RATE=20 \
    STRESS_TEST_TARGET_TPS=200 \
    STRESS_TEST=true

time go_test_junit_report e2e_against_$VCHAIN -timeout 10m -count=1 -v ./test/e2e/...
