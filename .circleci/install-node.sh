#!/bin/bash -e

curl -o- https://raw.githubusercontent.com/creationix/nvm/v0.33.11/install.sh | bash
export NVM_DIR="/home/circleci/.nvm" && . $NVM_DIR/nvm.sh && nvm install v11.2 && nvm use v11.2
