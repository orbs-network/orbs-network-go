#!/bin/bash -e

# Defined on CircleCI project-level, see https://circleci.com/gh/orbs-network/orbs-network-go/edit#env-vars
# MARVIN_ORCHESTRATOR_URL="http://ec2-34-222-245-15.us-west-2.compute.amazonaws.com:4567"
if [[ -z "${MARVIN_ORCHESTRATOR_URL}" ]] ; then
  echo "environment variable MARVIN_ORCHESTRATOR_URL must be defined"
  exit 1
fi

# Defined on job-level in .circleci/config.yml
if [[ -z "${JOB_ID}" ]] ; then
  echo "environment variable JOB_ID must be defined"
  echo "environment variable JOB_ID must be defined"
  exit 1
fi


# Get the JOB_ID from a file on the workspace

JOB_STATUS_URL="${MARVIN_ORCHESTRATOR_URL}/jobs/${JOB_ID}/status"
OUTPUT_FILE="workspace/job_status_${JOB_ID}.json"

mkdir -p workspace
curl "${JOB_STATUS_URL}" > "${OUTPUT_FILE}"
rc=$?
if [[ $rc -ne 0 ]] ; then
  echo "Failed to read status of job ${JOB_ID} from file ${OUTPUT_FILE}"
  exit 1
fi

# Can collect stdout into a file on the workspace and send it further
dredd --job_status "${OUTPUT_FILE}" > workspace/job_analysis_"${JOB_ID}"
rc=$?
if [[ ${rc} -ne 0 ]] ; then
  echo "Dredd has decided that Job ID ${JOB_ID} failed. Exit code: ${rc}"
  exit ${rc}
fi

echo "Dredd has decided that Job ID ${JOB_ID} was successful"


