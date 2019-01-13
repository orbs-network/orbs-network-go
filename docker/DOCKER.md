# Working with Docker

> Under construction: consider moving here all Docker-related docs, with ref from main README.md

All Orbs networks run on a Docker swarm, so from time to time you may want to test the code on real Docker containers in a swarm.

This folder contains scripts and configuration files for building and running tests on Docker.

## Troubleshooting

### Build errors

### Run errors

* `Bad response from Docker engine` - on Mac, make sure the Docker Desktop app is running and the marker is green when clicking the whale icon at the top
* `Pool overlaps with other one on this address space` - you need to clean up residual networks. First remove running containers with `docker rm -f $(docker ps -aq)`, then remove redundant networks with `docker network ls -q | xargs docker network rm`. 
You will see some errors when running this command as some networks cannot be deleted, this is normal.
* `Get https://registry-1.docker.io/v2/: proxyconnect tcp: dial tcp: lookup gateway.docker.internal on 192.168.199.1:53: read udp 192.168.199.1:50131->192.168.199.1:53: read: connection refused` - 
(or something similar) go to Docker Desktop's preferences, under Advanced, Make sure the Docker Subnet is not `192.168.199` (or the same address as your error shows)

 
