#!/bin/bash -xe

rm -rf logs
docker-compose -f docker-compose.acceptance.yml up --abort-on-container-exit --exit-code-from orbs-acceptance