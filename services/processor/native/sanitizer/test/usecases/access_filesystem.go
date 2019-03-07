package usecases

const AccessFilesystem = `package main

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"io/ioutil"
	"os"
)

var PUBLIC = sdk.Export(read)
var SYSTEM = sdk.Export(_init)

func _init() {
}

func read() {
	ioutil.ReadFile("/tmp/file")
	os.Open("/tmp/file2")
}
`
