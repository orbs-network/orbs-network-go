# Orbs Network

Orbs is a public blockchain infrastructure built for the needs of decentralized apps with millions of users. For more information, please check https://orbs.com and read the [white papers](https://orbs.com/white-papers).

This repo contains the node core reference implementation in golang.

The project is thoroughly tested with unit tests, component tests per microservice, acceptance tests, E2E tests and E2E stress tests running the system under load.

## Building from source

### Docker

If you only want to build the binaries, you don't need to have Golang on your machine. Having Docker is sufficient.

`./docker-build.sh` will create the images for you:
* `orbs:export` contains `orbs-node` and `orbs-json-client` binaries in `/opt/orbs` directory.
* `orbs:sambusac` contains self-sufficient development binary (similar to Ethereum's Ganache) and json client binary.

### Prerequisites

* Make sure [Go](https://golang.org/doc/install) is installed (version 1.10 or later).
  
  > Verify with `go version`

* Make sure [Go workspace bin](https://stackoverflow.com/questions/42965673/cant-run-go-bin-in-terminal) is in your path.
  
  > Install with ``export PATH=$PATH:`go env GOPATH`/bin``
  
  > Verify with `echo $PATH`

* Make sure Git is installed (version 2 or later).

  > Verify with `git --version`

### Build

* Clone the repo to your Go workspace:
```
cd `go env GOPATH`
go get github.com/orbs-network/orbs-network-go
cd src/github.com/orbs-network/orbs-network-go
git checkout master
```

* Install dependencies with `./git-submodule-checkout.sh`. To understand dependency management flow please refer to the [dependency documentation](DependencyManagement.md).

* Build with `go install`

* You can build all the binaries (`orbs-node`, `orbs-json-client` and `sambusac`) with `./build-binaries.sh`. All binaries are statically linked.

### Run

* To run the pre-built binary (should be in path):
```
orbs-network-go
```

* To rebuild from source and run (this will take you to project root):
```
cd `go env GOPATH`
cd src/github.com/orbs-network/orbs-network-go
go run *.go
```

## Testing from command line

### Available test runners

The official go test runner `go test` (has minimal UI and result caching). All tests can be run with `./test.sh`

### Test

* Run **all** tests from project root:

  * Using go test with `go test ./...`

* Run only **fast** tests (no E2E and similar):
  
  * Using go test with `go test -short ./...`
  
* Check unit test coverage:

  * Using go test with ``go test -cover `go list ./...` ``

### Test types

* Slow tests:

  ##### E2E tests

  > End-to-end tests check the entire system in a real life scenario mimicking real production with multiple nodes. It runs on docker with several nodes connected in a cluster. Due to their nature, E2E tests are slow to run.

  * The tests are found in [`/test/e2e`](test/e2e)
  * Run the suite from project root with `go test ./test/e2e`
  
  ##### Integration tests
  
  > Integration tests check the system adapters and make sure they meet the interface contract they implement. For example connection to a database or network sockets.

  * The tests are found per adapter (per service), eg. [`/services/gossip/adapter`](/services/gossip/adapter)

* Fast tests:

  ##### Acceptance tests

  > Acceptance tests check the internal hexagon of the system (it's logic with all microservices) with faster adapters that allow the suite to run extremely fast.  

  * The tests are found in [`/test/acceptance`](test/acceptance)
  * Run the suite from project root with `go test ./test/acceptance`

  ##### Component tests

  > Component tests check that a single service meets its specification while mocking the other services around it. They allow development of a service in isolation. 

  * The tests are found per service in the `test` directory, eg. [`/services/transactionpool/test`](/services/transactionpool/test)

  ##### Unit tests
  
  > Unit tests are very specific tests that check a single unit or two. They test the unit in isolation and stub/mock everything around it. 

  * The tests are found next to the actual unit in a file with `_test.go` suffix, eg. `sha256_test.go` sitting next to `sha256.go`

### Testing with Docker

Tests run automatically while we build Docker images because `test.sh` is part of the Docker build. `./docker-build.sh && ./docker-test.sh` will build all the images and then run e2e test in dockerized environment.

The logs for all e2e nodes are in `./logs` directory and will be deleted on every e2e run.

## Developer experience

### Debugging with Docker

`./docker-build.debug.sh` and `docker-test.debug.sh` provide shorter development cycle by skipping tests and avoiding building development tools while building the image. The purpose is to let developers run e2e as soon as possible because some of the issues only manifest inside Docker.

If the e2e gets stuck or `docker-compose` stops working properly, try to **remove all containers** with this handy command: `docker rm -f $(docker ps -aq)`. But remember that **ALL YOUR CONTAINERS WILL BE GONE**. All of them.

### IDE

* We recommend working on the project with [GoLand](https://www.jetbrains.com/go/) IDE. Recommended settings:
  * Under `Preferences | Editor | Code Style | Go` make sure `Use tab character` is checked

* For easy testing, under `Run | Edit Configurations` add these `Go Test` configurations:
  * "Fast" with `Directory` set to project root and `-short` flag added to `Go tool arguments`
  * "All" with `Directory` set to project root
  * It's also recommended to uncheck `Show Ignored` tests and check `Show Passed` in the test panel after running the configuration
  * If you have a failed test which keeps failing due to cache click `Rerun Failed Tests` in the test panel (it will ignore cache)

* You may enable the following automatic tools that run on file changes:
  * "go fmt" in `Preferences | Tools | File Watchers`, add with `+` the `go fmt` watcher
  * To run tests automatically on save, check `Toggle auto-test` in the test panel (it's now a core feature of GoLand)

* Debugging tests may contain very verbose logs, increase console buffer size in `Preferences | Editor | General | Console | Override console cycle buffer size = 10024 KB`

* If you experience lags while working with GoLand, increasing its default VM heap size can help:
 * Go to `Help | Edit Custom VM Options...` and set:
 ```
 -Xms256m
 -Xmx1536m
 ```

## License

MIT
