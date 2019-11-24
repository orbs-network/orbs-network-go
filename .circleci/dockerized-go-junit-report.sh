#!/bin/bash -e

$(aws ecr get-login --no-include-email --region us-west-2)

docker run -i 727534866935.dkr.ecr.us-west-2.amazonaws.com/go-junit-report:latest $@
