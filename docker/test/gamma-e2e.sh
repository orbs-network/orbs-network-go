#!/bin/bash -x

. ./test.common.sh

echo "Cleaning up all containers, if any are running"
# This is here so that in CI we would be able to see what was cleaned.
docker ps -a
echo "Cleaned the following containers:"
(docker ps -aq | xargs docker rm -fv) || echo "No containers to clean! Good!"
sleep 3

echo "Spinning a Gamma instance.."
docker-compose -f ./docker/test/docker-compose-gamma.yml up -d
EXIT_CODE=$?
if [ $EXIT_CODE -ne 0 ]
  then exit $EXIT_CODE
fi

export API_ENDPOINT=http://localhost:8080

echo "Running Gamma server tests with Ganache.."
time go_test_junit_report gamma-e2e -count=1 ./bootstrap/gamma/e2e/...

docker-compose -f ./docker/test/docker-compose-gamma.yml down
