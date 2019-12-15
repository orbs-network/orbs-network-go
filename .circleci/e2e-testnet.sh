#!/bin/bash

. ./test.common.sh

# Get the vchains to act upon from CircleCI's workspace
VCHAIN=$(cat workspace/app_chain_id)
MGMT_VCHAIN=$(cat workspace/mgmt_chain_id)
TESTNET_IP=$(cat workspace/testnet_ip)

echo "Running E2E on deployed app chain ($VCHAIN) and management chain ($MGMT_VCHAIN)"
echo "on IP: $TESTNET_IP"

export API_ENDPOINT=http://$TESTNET_IP/vchains/$VCHAIN/ \
    MGMT_API_ENDPOINT=http://$TESTNET_IP/vchains/$MGMT_VCHAIN/ \
    REMOTE_ENV="true" \
    STRESS_TEST_NUMBER_OF_TRANSACTIONS=100 \
    STRESS_TEST_FAILURE_RATE=20 \
    STRESS_TEST_TARGET_TPS=200 \
    STRESS_TEST=true

env

time go_test_junit_report e2e_against_$VCHAIN -timeout 10m -count=1 -v ./test/e2e/...
