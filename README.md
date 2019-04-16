# Orbs Network

[![CI](https://circleci.com/gh/orbs-network/orbs-network-go/tree/master.svg?style=svg)](https://circleci.com/gh/orbs-network/orbs-network-go/tree/master)

Orbs is a public blockchain infrastructure built for the needs of decentralized apps with millions of users. For more information, please check https://orbs.com and read the [white papers](https://orbs.com/white-papers).

This repo contains the node core reference implementation in golang.

The project is thoroughly tested with unit tests, component tests per microservice, acceptance tests, E2E tests and E2E stress tests running the system under load.

## Building Docker images only

If you only want to build the Docker images containing the node binaries, you don't need to have golang on your own machine (the node will be compiled inside the image).

* Make sure Docker is [installed](https://docs.docker.com/install/).

  > Verify with `docker version`

* Run `./docker/build/build.sh` to create the images:

  * `orbs:export` contains `orbs-node` and `gamma-cli` binaries in the `/opt/orbs` directory.
  * `orbs:gamma-server` contains self-sufficient development binary (similar to Ethereum's Ganache) and `gamma-cli` to communicate with it's server counterpart.

## Building from source

### Prerequisites

* Make sure [Go](https://golang.org/doc/install) is installed (version 1.10 or later).
  
  > Verify with `go version`

* Make sure [Go workspace bin](https://stackoverflow.com/questions/42965673/cant-run-go-bin-in-terminal) is in your path.
  
  > Install with ``export PATH=$PATH:`go env GOPATH`/bin``
  
  > Verify with `echo $PATH`

* Make sure Git is installed (version 2 or later).

  > Verify with `git --version`

* If you're interested in building Docker images as well, install [Docker](https://docs.docker.com/install/).

  > Verify with `docker version`

### Build

* Clone the repo to your Go workspace:
  ```
  cd `go env GOPATH`
  go get github.com/orbs-network/orbs-network-go
  cd src/github.com/orbs-network/orbs-network-go
  git checkout master
  ```

* Install dependencies with `./git-submodule-checkout.sh`. To understand dependency management flow please refer to the [dependency documentation](DEPENDENCIES.md).

* Build with `go install`

* You can build all the binaries (`orbs-node`, `gamma-cli` and `gamma-server`) with `./build-binaries.sh`.

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

### Test runner

We use the official go test runner `go test`. It has minimal UI and result caching.

Please install go-junit-reporter prior to running tests for the first time:
```
go get -u github.com/orbs-network/go-junit-report
```

### Test

* Run **all** tests using a script:

    `./test.sh`

* Manually run **all** tests from project root:

    `go test ./...`

* Manually run only **fast** tests (no E2E and similar):

    `go test -short ./...`
  
* Check unit test coverage:

    ``go test -cover `go list ./...` ``

### Test types

#### E2E tests (slow)

> End-to-end tests check the entire system in a real life scenario mimicking real production with multiple nodes. It runs on docker with several nodes connected in a cluster. Due to their nature, E2E tests are slow to run.

* The tests are found in [`/test/e2e`](test/e2e)
* Run the suite from project root with `go test ./test/e2e`
  
#### Integration tests (slow)

> Integration tests check the system adapters and make sure they meet the interface contract they implement. For example connection to a database or network sockets.

* The tests are found per adapter (per service), eg. [`/services/gossip/adapter`](/services/gossip/adapter)

#### Acceptance tests (fast)

> Acceptance tests check the internal hexagon of the system (it's logic with all microservices) with faster adapters that allow the suite to run extremely fast.

* The tests are found in [`/test/acceptance`](test/acceptance)
* Run the suite from project root with `go test ./test/acceptance`

#### Component tests (fast)

> Component tests check that a single service meets its specification while mocking the other services around it. They allow development of a service in isolation.

* The tests are found per service in the `test` directory, eg. [`/services/transactionpool/test`](/services/transactionpool/test)

#### Unit tests (fast)

> Unit tests are very specific tests that check a single unit or two. They test the unit in isolation and stub/mock everything around it.

* The tests are found next to the actual unit in a file with `_test.go` suffix, eg. `sha256_test.go` sitting next to `sha256.go`

### Testing on Docker

> For Troubleshooting, see the [Docker Guide](docker/docker.md)

All tests run automatically when the Docker images are built. The script `./test.sh` is part of the Docker build. 

* Run `./docker/build/build.sh && ./docker/test/test.sh` to build all images and run E2E tests in a dockerized environment.

* The logs for all E2E nodes will be placed on your machine under the `./_logs` directory of the project (and will be overridden on every E2E run).

#### Component tests on Docker
To detect flaky tests of specific components, run **component** tests multiple times on Docker:
* To enable running component tests multiple times, edit `test.sh` and uncomment the line `./test.components.sh`
* To modify the number of times each component test runs, edit `test.components.sh` and modify the value of the **COUNT** variable (you may also need to modify the various timeouts)
* *optional* To run a specific test (component, unit, of any other) multiple times (useful when debugging a specific scenario for flakiness), edit `test.components.sh`, comment the line `run_component_tests`,
then uncomment the line starting with `run_specific_test` and modify the test name to run that one specific test multiple times on Docker.

After you've finished editing, run `./docker/build/build.sh && ./docker/test/test.sh`

> You should probably not commit any of these edits you've made for testing, as they are transient in nature.

## Developer experience

### Git hooks

Please run `git config --local core.hooksPath .githooks` after cloning the repository.

### Debugging issues on Docker

Occasionally, local tests with `go test` will pass but the same tests on Docker will fail. This usually happens when tests are flaky and sensitive to timing (we do our best to avoid this). 

* Run `./docker/build/build.sh` and `./docker/test/test.sh`.

* If the E2E test gets stuck or `docker-compose` stops working properly, try to **remove all containers** with this handy command: `docker rm -f $(docker ps -aq)`. But remember that **ALL YOUR LOCAL CONTAINERS WILL BE GONE** (even from other projects).

* Debugging the acceptance suite is problematic out of the box, since the debugger (Delve) doesn't support any code importing the `plugin` package. Luckily, the acceptance suite relies on a fake compiler; to enable debugging:
  * in Goland
    1. Go to Preferences -> Go -> Vendoring & Build Tags
    1. Add the tag `nonativecompiler` under 'custom tags'
    1. Create a run configuration for the desired test (by clicking the "play" icon to the left of the test name)
    1. Run it once (to create the run config)
    1. Edit the run config and check "use all custom build tags"
    1. Debug your test

### IDE

* We recommend working on the project with [GoLand](https://www.jetbrains.com/go/) IDE. Recommended settings:
  * Under `Preferences | Editor | Code Style | Go` make sure `Use tab character` is checked

* For easy testing, under `Run | Edit Configurations` add these `Go Test` configurations:
  * "Fast" with `Directory` set to project root and `-short` flag added to `Go tool arguments`
  * "All" with `Directory` set to project root
  * It's also recommended to uncheck `Show Ignored` tests and check `Show Passed` in the test panel after running the configuration
  * If you have a failed test which keeps failing due to cache click `Rerun Failed Tests` in the test panel (it will ignore cache)

* Running some tests that are unsafe for production deployments requires a special build flag, enable it if you're a core developer:
  * Under `Preferences | Go | Vendoring & Build Tags | Custom tags ` add the tag `unsafetests`

* You may enable the following automatic tools that run on file changes:
  * "go fmt" in `Preferences | Tools | File Watchers`, add with `+` the `go fmt` watcher
  * To run tests automatically on save, check `Toggle auto-test` in the test panel (it's now a core feature of GoLand)

* Debugging tests may contain very verbose logs, increase console buffer size in `Preferences | Editor | General | Console | Override console cycle buffer size = 10024 KB`

* If you experience lags or Low Memory warnings while working with GoLand, increasing its default VM heap size can help:
 * Go to `Help | Edit Custom VM Options...` and set:
   ```
   -Xms256m
   -Xmx1536m
   ```

### Profiling

To enable profiling: put `"profiling": true` in your `config.json`.

It will enable [net/http/pprof](https://golang.org/pkg/net/http/pprof/) package, and you will be able to query `pprof` via http just as described in the docs.

### Debugging with logs

By default, log output is filtered to only errors and metrics. To enable full log, put `"logger-full-log": true` in your node configuration. It will permanently remove the filter.

If you want to enable or disable this filter in production, there is a way to do that via HTTP API:

```
curl -XPOST http://$NODE_IP/vchains/$VCHAIN/debug/logs/fiter-on
```

Or

```
curl -XPOST http://$NODE_IP/vchains/$VCHAIN/debug/logs/fiter-off
```

## Development principles
Refer to the [Contributor's Guide](CONTRIBUTING.md) (work in progress)


## License

MIT
