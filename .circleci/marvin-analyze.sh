#!/bin/bash -e

# Get the JOB_ID from a file on the workspace
JOB_ID=20191215_172942_073

# Can collect stdout into a file on the workspace and send it further
dredd --job_id ${JOB_ID}
rc=$?

if [[ ${rc} -eq 0 ]] ; then
  echo "Dredd has decided that Job ID ${JOB_ID} was successful"
else
  echo "Dredd has decided that Job ID ${JOB_ID} failed. Exit code: ${rc}"
fi
