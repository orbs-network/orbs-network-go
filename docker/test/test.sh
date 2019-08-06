#!/bin/bash -e
rm -rf _logs

# Important note: trying to run the stress test locally? you will need to increase your max allowed sockets open / open files
# as shown in this stack overflow URL:
# https://stackoverflow.com/questions/7578594/how-to-increase-limits-on-sockets-on-osx-for-load-testing

[[ -z $CONSENSUSALGO ]] && echo "Consensus algo is not set! quiting.." && exit 1
#docker-compose -f ./docker/test/docker-compose.yml down

export GIT_BRANCH=$(source ./docker/tag.sh)
export GIT_COMMIT=$(git rev-parse HEAD)
export SRC=/go/src/github.com/orbs-network/orbs-network-go

# prepare persistent blocks for docker tests

rm -rf _tmp/blocks
mkdir -p _tmp/blocks/node{1..4}

cp ./test/e2e/_data/blocks _tmp/blocks/node1
cp ./test/e2e/_data/blocks _tmp/blocks/node2
cp ./test/e2e/_data/blocks _tmp/blocks/node3
cp ./test/e2e/_data/blocks _tmp/blocks/node4

# run docker-reliant tests
docker-compose -f ./docker/test/docker-compose.yml up -d

export API_ENDPOINT=http://localhost:8080/api/v1/ \
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
echo "Sleeping to allow the network to start closing new blocks.."
echo "(So that the txpool won't throw our calls to the bin)"
sleep 10

echo "Running E2E tests w/consensus algo: ${CONSENSUSALGO}"
go test -count=1 -v ./test/e2e/...

echo "Running Ethereum Connector tests w/consensus algo: ${CONSENSUSALGO}"
go test -count=1 -v ./test/ethereum/...

echo "Running Gamma tests w/consensus algo: ${CONSENSUSALGO}"
go test -count=1 -v ./bootstrap/gamma/e2e/...
