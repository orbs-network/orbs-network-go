// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"os"
)

func main() {
	logger := instrumentation.GetBootstrapCrashLogger()
	var node *bootstrap.Node
	func() { // context of bootstrap crash logging
		defer func() {
			if r := recover(); r != nil {
				logger.Error("unexpected error during bootstrap", log.Error(errors.Errorf("unknown error: %v", r)))
				os.Exit(8)
			}
		}()
		httpAddress := flag.String("listen", ":8080", "ip address and port for http server")
		silentLog := flag.Bool("silent", false, "disable output to stdout")
		pathToLog := flag.String("log", "", "path/to/node.log")
		version := flag.Bool("version", false, "returns information about version")

		var configFiles config.ArrayFlags
		flag.Var(&configFiles, "config", "path/to/config.json")

		flag.Parse()

		if *version {
			fmt.Println(config.GetVersion())
			return
		}

		cfg, err := config.GetNodeConfigFromFiles(configFiles, *httpAddress)
		if err != nil {
			logger.Error("error reading configuration", log.Error(err))
			os.Exit(1)
		}

		logger = instrumentation.GetLogger(*pathToLog, *silentLog, cfg)

		node = bootstrap.NewNode(
			cfg,
			logger,
		)

		supervised.NewShutdownListener(logger, node).ListenToOSShutdownSignal()
	}()
	defer func() {
		if r := recover(); r != nil {
			logger.Error("unexpected error in main goroutine", log.Error(errors.Errorf("unknown error: %v", r)))
			os.Exit(2)
		}
	}()
	node.WaitUntilShutdown(context.Background())
}
