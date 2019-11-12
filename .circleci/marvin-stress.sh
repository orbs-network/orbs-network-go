#!/bin/bash -e

echo "Launching Marvin stress test"

URI="ec2-34-222-245-15.us-west-2.compute.amazonaws.com:4567/jobs/start"
curl -d '{"tpm":60, "duration_sec":720, "client_timeout_sec": 180}' -H "Content-Type: application/json" -X POST ${URI}
echo
echo "Started Marvin test. Results will be posted to Slack channel #marvin-results."
echo

