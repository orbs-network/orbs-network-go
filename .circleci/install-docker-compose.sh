#!/bin/bash -e

echo "Installing a newer version of Docker Compose.."

COMPOSE_VERSION="1.24.1"

curl -L "https://github.com/docker/compose/releases/download/$COMPOSE_VERSION/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose
docker-compose --version