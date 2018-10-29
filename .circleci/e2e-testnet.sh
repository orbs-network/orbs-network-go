#!/usr/bin/env bash

docker run -ti \
    -e API_ENDPOINT=http://us-east-1.global.nodes.staging.orbs-test.com/api/v1/ \
    -e STRESS_TEST_NUMBER_OF_TRANSACTIONS=10000 \
    -e STRESS_TEST_FAILURE_RATE=20 \
    -e STRESS_TEST_TARGET_TPS=250 \
    -e STRESS_TEST=true \
    orbs:e2e