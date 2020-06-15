#!/bin/bash -xe

# Important note: trying to run the stress test locally? you will need to increase your max allowed sockets open / open files
# as shown in this stack overflow URL:
# https://stackoverflow.com/questions/7578594/how-to-increase-limits-on-sockets-on-osx-for-load-testing

. ./test.common.sh

echo "Cleaning up all containers, if any are running"
docker ps -a
echo "Cleaned the following containers:"
(docker ps -aq | xargs docker rm -fv) || echo "No containers to clean! Good!"
sleep 3

export NVM_DIR="/opt/circleci/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"  # This loads nvm bash_completion
nvm install v11.2 && nvm use v11.2
echo "Using node: "
node -v
cd .circleci && npm install && cd ..

rm -rf _logs _out

[[ -z $CONSENSUSALGO ]] && echo "Consensus algo is not set! quiting.." && exit 1

export GIT_BRANCH=$(source ./docker/tag.sh)
export GIT_COMMIT=$(git rev-parse HEAD)
export SRC=/go/src/github.com/orbs-network/orbs-network-go

# prepare persistent blocks for docker tests
# this is a weird trick to get around filesystem permissions to avoid using sudo and blocking things on Mac
docker run --rm -ti -v $(pwd)/_tmp:/opt/_tmp busybox sh -c "rm -rf /opt/_tmp/* && mkdir -p /opt/_tmp/blocks/node{1..4} && chmod -R 0777 /opt/_tmp/"

# We do not copy blocks for node1 to check the block sync
cp ./test/e2e/_data/blocks _tmp/blocks/node1
cp ./test/e2e/_data/blocks _tmp/blocks/node2
cp ./test/e2e/_data/blocks _tmp/blocks/node3
#cp ./test/e2e/_data/blocks _tmp/blocks/node4

# run docker-reliant tests
docker-compose -f ./docker/test/docker-compose.yml up -d
EXIT_CODE=$?
if [ $EXIT_CODE -ne 0 ]
  then exit $EXIT_CODE
fi

export API_ENDPOINT=http://localhost:8082/api/v1/ \
      VCHAIN=42 \
      STRESS_TEST_NUMBER_OF_TRANSACTIONS=5000 \
      STRESS_TEST_FAILURE_RATE=20 \
      STRESS_TEST_TARGET_TPS=100 \
      STRESS_TEST='true' \
      ETHEREUM_ENDPOINT=http://localhost:8545/ \
      ETHEREUM_PRIVATE_KEY=f2ce3a9eddde6e5d996f6fe7c1882960b0e8ee8d799e0ef608276b8de4dc7f19
      ETHEREUM_PUBLIC_KEY=037a809cc481303d337c1c83d1ba3a2222c7b1b820ac75e3c6f8dc63fa0ed79b18 \
      EXTERNAL_TEST='true'

# the ethereum keypair is generated from the mnemonic passed to ganache on startup

echo "The network has started with pre-existing (ancient) 500-some blocks"

echo "Polling the app network for liveness.."
./.circleci/check-e2e-network-liveness.js 42 10

#echo "Polling the management network for liveness.."
#./.circleci/check-e2e-network-liveness.js 40 10

echo "Running E2E tests (AND a humble stress-test) w/consensus algo: ${CONSENSUSALGO}"
time go_test_junit_report e2e -timeout 10m -count=1 ./test/e2e/...
