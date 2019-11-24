#!/bin/bash -e

$(aws ecr get-login --no-include-email --region us-west-2)

docker build -f ./docker/build/Dockerfile.go_junit_report -t 727534866935.dkr.ecr.us-west-2.amazonaws.com/go-junit-report:latest .
docker push 727534866935.dkr.ecr.us-west-2.amazonaws.com/go-junit-report:latest
