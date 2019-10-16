#!/bin/bash -x

. ./test.common.sh

echo "Cleaning up all containers, if any are running"
# This is here so that in CI we would be able to see what was cleaned.
docker ps -a
echo "Cleaned the following containers:"
(docker ps -aq | xargs docker rm -fv) || echo "No containers to clean! Good!"
sleep 3

echo "Spinning a Ganache instance.."
docker-compose -f ./docker/test/docker-compose-ganache.yml up -d

export ETHEREUM_ENDPOINT=http://localhost:8545/ \
      ETHEREUM_PRIVATE_KEY=f2ce3a9eddde6e5d996f6fe7c1882960b0e8ee8d799e0ef608276b8de4dc7f19

echo "Running cross chain connector tests with Ganache.."
time go_test_junit_report crosschainconnector -timeout 10m -count=1 ./services/crosschainconnector/ethereum/...

echo "Running Gamma server tests with Ganache.."
time go_test_junit_report gamma -timeout 10m -count=1 ./bootstrap/gamma/ethereum/...

