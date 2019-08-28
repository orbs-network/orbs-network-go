#!/bin/bash -x

export VCHAIN=$1
export MGMT_VCHAIN=$2

export API_ENDPOINT=http://35.172.102.63/vchains/$VCHAIN/ \
    MGMT_API_ENDPOINT=http://35.172.102.63/vchains/$MGMT_VCHAIN/ \
    REMOTE_ENV="true" \
    STRESS_TEST_NUMBER_OF_TRANSACTIONS=100 \
    STRESS_TEST_FAILURE_RATE=20 \
    STRESS_TEST_TARGET_TPS=200 \
    STRESS_TEST=true \

go test ./test/e2e/... -v
