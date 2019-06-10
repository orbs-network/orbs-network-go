#!/bin/bash -e

# Determine if we have an active PR
if [ -z "$CI_PULL_REQUESTS" ]
then
    echo "We have an active PR ($CI_PULL_REQUESTS)"
else
    echo "No active PR, exiting.."
fi

exit 0