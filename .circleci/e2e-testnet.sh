#!/bin/bash -x

export VCHAIN=$1

export API_ENDPOINT=http://us-east-1.global.nodes.staging.orbs-test.com/vchains/$VCHAIN/api/v1/ \
    STRESS_TEST_NUMBER_OF_TRANSACTIONS=100 \
    STRESS_TEST_FAILURE_RATE=20 \
    STRESS_TEST_TARGET_TPS=200 \
    STRESS_TEST=true \

go test ./test/e2e/... -v
