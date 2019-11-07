#!/bin/bash -e

echo "Launching Marvin stress test"

URI="34.222.245.15:4567/jobs/start"

curl -d '{"tpm":10, "duration_sec":60}' -H "Content-Type: application/json" -X POST ${URI}

echo "Finished Marvin stress test"

