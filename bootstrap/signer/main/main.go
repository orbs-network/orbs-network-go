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
	"github.com/orbs-network/orbs-network-go/bootstrap/signer"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/scribe/log"
	"os"
)

func getLogger() log.Logger {
	return log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
}

func main() {
	httpAddress := flag.String("listen", ":7777", "ip address and port for http server")
	version := flag.Bool("version", false, "returns information about version")

	var configFiles config.ArrayFlags
	flag.Var(&configFiles, "config", "path/to/config.json")

	flag.Parse()

	if *version {
		fmt.Println(config.GetVersion())
		return
	}

	logger := getLogger()
	cfg, err := config.GetNodeConfigFromFiles(configFiles, *httpAddress)
	if err != nil {
		logger.Error("failed to parse configuration", log.Error(err))
		os.Exit(1)
	}

	server, err := signer.StartSignerServer(cfg, logger)
	if err != nil {
		logger.Error("failed to start signer service", log.Error(err))
		os.Exit(1)
	}

	server.WaitUntilShutdown(context.Background())
}
