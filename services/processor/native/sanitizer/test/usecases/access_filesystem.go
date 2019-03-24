// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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
