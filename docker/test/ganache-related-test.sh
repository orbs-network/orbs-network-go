#!/bin/bash -e

. ./test.common.sh

echo "Cleaning up all containers, if any are running"
# This is here so that in CI we would be able to see what was cleaned. 
docker ps -a
echo "Cleaned the following containers:"
(docker ps -aq | xargs docker rm -fv) || echo "No containers to clean! Good!"
sleep 3

echo "Spinning a Ganache instance.."
docker-compose -f ./docker/test/docker-compose-ganache.yml up -d

echo "Running cross chain connector tests with Ganache.."
time go_test_junit_report crosschainconnector -timeout 10m -count=1 ./services/crosschainconnector/ethereum/adapter/...

echo "Running Gamma server tests with Ganache.."
time go_test_junit_report gamma -timeout 10m -count=1 ./bootstrap/gamma/e2e/...

