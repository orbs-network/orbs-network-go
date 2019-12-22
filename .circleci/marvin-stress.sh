#!/bin/bash -e

echo "Launching Marvin stress test"

# Defined on CircleCI project-level, see https://circleci.com/gh/orbs-network/orbs-network-go/edit#env-vars
# MARVIN_ORCHESTRATOR_URL="http://ec2-34-222-245-15.us-west-2.compute.amazonaws.com:4567"
if [[ -z "${MARVIN_ORCHESTRATOR_URL}" ]] ; then
  echo "environment variable MARVIN_ORCHESTRATOR_URL must be defined"
  exit 1
fi

if [[ -z "${APP_CHAIN_ID}" ]] ; then
  echo "environment variable MARVIN_ORCHESTRATOR_URL must be defined"
  exit 1
fi


ORCH_STATUS_URL="${MARVIN_ORCHESTRATOR_URL}/status"
curl "${ORCH_STATUS_URL}"
rc=$?
if [[ ${rc} -ne 0 ]] ; then
  echo "Failed to check Marvin orchestrator status. Process is probably down. Login to user ubuntu on marvin machine ${MARVIN_ORCHESTRATOR_URL} and run 'pm2 list'."
  exit 1
fi

URI="${MARVIN_ORCHESTRATOR_URL}/jobs/start"

curl -d "{\"client_timeout_sec\": 10, \"vchain\": ${APP_CHAIN_ID}, \"target_ips\": [\"35.161.123.97\"]}" -H "Content-Type: application/json" -X POST ${URI}
echo "Started Marvin test. Results will be posted to Slack channel #marvin-results."
echo

