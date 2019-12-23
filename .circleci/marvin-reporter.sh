#!/bin/bash -e

# If running locally, need to disable these next 4 lines
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"                   # This loads nvm
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion" # This loads nvm bash_completion
nvm use "${NODE_VERSION}"

cd .circleci && npm install && cd ..

./.circleci/marvin-reporter.js "$1"