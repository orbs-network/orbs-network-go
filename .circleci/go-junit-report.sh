#!/bin/bash

if [ $USE_DOCKERIZED_JUNIT_REPORT == "true" ]
  $(aws ecr get-login --no-include-email --region us-west-2) >/dev/null 2>&1
  docker run -i 727534866935.dkr.ecr.us-west-2.amazonaws.com/go-junit-report:latest $@
else
  go-junit-report $@
fi


