# Enabling JavaScript Processor Experiment

* Install and compile `github.com/ry/v8worker2`

  * Run `brew install pkg-config`

  * Run `go get github.com/ry/v8worker2`
  
  * Run ``cd `go env GOPATH`/src/github.com/ry/v8worker2`` and then `./build.py` (will take ~30 min)

* Enable the experiment by passing `-tags jsprocessor` to the relevant go tool.

    For example `go build -tags jsprocessor ...` or `go test -tags jsprocessor ...`