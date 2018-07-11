# Orbs Network

Orbs is a public blockchain infrastructure built for the needs of decentralized apps with millions of users. For more information, please check https://orbs.com and read the [white papers](https://orbs.com/white-papers).

This repo contains the node core reference implementation in golang.

The project is thoroughly tested with unit tests, component tests per microservice, acceptance tests, E2E tests and E2E stress tests running the system under load.

## Building from source

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
git checkout master // or dev if you want the dev branch
```

* Install dependencies with `go get -t ./...`

* Build with `go install`

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

### Prerequisites for tests

* Make sure [Ginkgo](http://onsi.github.io/ginkgo/#getting-ginkgo) is installed.
  
  > Install with `go get github.com/onsi/ginkgo/ginkgo`
  
  > Verify with `ginkgo version`

### Test

* Run **all** tests from project root with `ginkgo ./...`

* Another alternative runner with minimal UI and result caching:

  * Run **all** tests with `go test ./...`
  
* Check unit test coverage with:
    ```
    go test -cover `go list ./...`
    ```

##### E2E tests

> End-to-end tests check the entire system in a real life scenario mimicking real production. It runs on docker with several nodes connected in a cluster. Due to their nature, E2E tests are slow to run.

* The tests are found in [`/test/e2e`](test/e2e)
* Run the suite from project root with `ginkgo -v ./test/e2e`

##### Acceptance tests

> Acceptance tests check the internal hexagon of the system (it's logic with all microservices) with faster adapters that allow the suite to run extremely fast.  

* The tests are found in [`/test/acceptance`](test/acceptance)
* Run the suite from project root with `ginkgo -v ./test/acceptance`

## Developer experience

### IDE

* We recommend working on the project with [GoLand](https://www.jetbrains.com/go/) IDE.

* For easy testing, under `Run - Edit Configurations` add these `Go Test` configurations:
  * "Acceptance" with `File` set to `your-path-to/orbs-network-go/test/acceptance/acceptance_test.go`
  * "E2E" with `File` set to `your-path-to/orbs-network-go/test/e2e/e2e_test.go`

## License

MIT