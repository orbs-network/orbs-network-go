#!/bin/bash -e

JOB_ID=$(cat workspace/job_id)

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

# If running locally, need to disable these next 4 lines
if [[ "$CI" == "true" ]]; then 
  sudo apt-get install -y gnupg
  gpg --yes --batch --passphrase="${MARVIN_PRIVATE_KEY_SECRET}" .circleci/marvin/marvin.pem.gpg

  export NVM_DIR="$HOME/.nvm"
  [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"                   # This loads nvm
  [ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion" # This loads nvm bash_completion
  nvm use "${NODE_VERSION}"
fi

npm install

# Get the JOB_ID from a file on the workspace

JOB_STATUS_URL="${MARVIN_ORCHESTRATOR_URL}/jobs/${JOB_ID}/status"
JOB_RESULTS_FILE="results.json"

curl "${JOB_STATUS_URL}" > "${JOB_RESULTS_FILE}"

LAST_MASTER_JOBS_URL="${MARVIN_ORCHESTRATOR_URL}/jobs/list/all/transferFrenzy/branch/master"
LAST_MASTERS_FILE="last_masters.json"

curl "${LAST_MASTER_JOBS_URL}" > "${LAST_MASTERS_FILE}"

# Probably the job was not found, just print the curl result
if [[ $(cat "${JOB_RESULTS_FILE}" | grep -c "not found") -ne 0 ]] ; then
  cat "${JOB_RESULTS_FILE}"
  exit 2
fi

# Can collect stdout into a file on the workspace and send it further
node .circleci/marvin/marvin-analyze.js "../../${JOB_RESULTS_FILE}" "../../${LAST_MASTERS_FILE}"

echo "Job analysis complete. Results written to ../../${JOB_RESULTS_FILE}"