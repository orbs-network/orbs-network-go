#!/bin/bash

mkdir -p ~/.aws

echo "$(cat <<-EOF
[default]
aws_access_key_id = $AWS_ACCESS_KEY_ID
aws_secret_access_key = $AWS_SECRET_ACCESS_KEY
EOF
)" > ~/.aws/credentials
