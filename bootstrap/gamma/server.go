// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package gamma

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/bootstrap/httpserver"
	"github.com/orbs-network/orbs-network-go/bootstrap/inmemory"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/scribe/log"
)

type GammaServer struct {
	bootstrap.OrbsProcess
	network *inmemory.Network
}

type GammaServerConfig struct {
	ServerAddress      string
	Profiling          bool
	OverrideConfigJson string
	Silent             bool
}

func getLogger(silent bool) log.Logger {

	if silent {
		return log.GetLogger().WithOutput()
	}

	return log.GetLogger().
		WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter())).
		WithFilters(
			//TODO(https://github.com/orbs-network/orbs-network-go/issues/585) what do we really want to output to the gamma server log? maybe some meaningful data for our users?
			log.IgnoreMessagesMatching("state transitioning"),
			log.IgnoreMessagesMatching("finished waiting for responses"),
			log.IgnoreMessagesMatching("no responses received"),
		)
}

func StartGammaServer(config GammaServerConfig) *GammaServer {
	ctx, cancel := context.WithCancel(context.Background())
	rootLogger := getLogger(config.Silent)

	network := NewDevelopmentNetwork(ctx, rootLogger, config.OverrideConfigJson)
	rootLogger.Info("finished creating development network")

	httpServer := httpserver.NewHttpServer(httpserver.NewServerConfig(config.ServerAddress, config.Profiling),
		rootLogger, network.PublicApi(0), network.MetricRegistry(0))

	s := &GammaServer{
		OrbsProcess: bootstrap.NewOrbsProcess(rootLogger, cancel, httpServer),
		network:     network,
	}

	s.addGammaHandlers(httpServer.Router())

	return s
}

func (n *GammaServer) Port() int {
	return n.HttpServer.Port()
}

var (
	port               = flag.Int("port", 8080, "The port to bind the gamma server to")
	profiling          = flag.Bool("profiling", false, "enable profiling")
	version            = flag.Bool("version", false, "returns information about version")
	overrideConfigJson = flag.String("override-config", "{}", "JSON-formatted config overrides, same format as the file config")
)

func Main() {

	flag.Parse()

	if *version {
		fmt.Println(config.GetVersion())
		return
	}

	var serverAddress = ":" + strconv.Itoa(*port)

	StartGammaServer(GammaServerConfig{
		ServerAddress:      serverAddress,
		Profiling:          *profiling,
		OverrideConfigJson: *overrideConfigJson,
		Silent:             false,
	}).WaitUntilShutdown()
}
