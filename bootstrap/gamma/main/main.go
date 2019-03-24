// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package main

import (
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap/gamma"
	"github.com/orbs-network/orbs-network-go/config"
	"strconv"
)

func main() {
	port := flag.Int("port", 8080, "The port to bind the gamma server to")
	profiling := flag.Bool("profiling", false, "enable profiling")
	version := flag.Bool("version", false, "returns information about version")
	overrideConfigJson := flag.String("override-config", "{}", "JSON-formatted config overrides, same format as the file config")

	flag.Parse()

	if *version {
		fmt.Println(config.GetVersion())
		return
	}

	var serverAddress = ":" + strconv.Itoa(*port)

	// TODO(v1) add WaitUntilShutdown so this behaves like the regular main (no blocking flag)
	gamma.StartGammaServer(serverAddress, *profiling, *overrideConfigJson, true)
}
