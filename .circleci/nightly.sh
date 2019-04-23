#!/bin/bash

echo "Installing nightly dependencies.."
go get -u github.com/orbs-network/go-junit-report
curl -o- https://raw.githubusercontent.com/creationix/nvm/v0.33.11/install.sh | bash
export NVM_DIR="/home/circleci/.nvm" && . $NVM_DIR/nvm.sh && nvm install v11.2 && nvm use v11.2
npm install junit-xml-stats -g

echo "Running the nightly suite"
./test.flakiness.sh NIGHTLY