#!/bin/bash -e

JOB_ANALYSIS_FILE="$1"

if [[ -z "${JOB_ANALYSIS_FILE}" ]] ; then
  echo "Job analysis file was not provided"
  exit 1
fi

# If running locally, need to disable these next 4 lines
if [[ "$CI" == "true" ]]; then
  export NVM_DIR="$HOME/.nvm"
  [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"                   # This loads nvm
  [ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion" # This loads nvm bash_completion
  nvm use "${NODE_VERSION}"
fi

npm install

./.circleci/marvin/marvin-reporter.js "${JOB_ANALYSIS_FILE}"