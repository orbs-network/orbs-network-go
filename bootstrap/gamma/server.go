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
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/services/transactionpool/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"os"
	"strconv"

	"github.com/orbs-network/orbs-network-go/bootstrap/httpserver"
	"github.com/orbs-network/orbs-network-go/bootstrap/inmemory"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/scribe/log"
)

type Server struct {
	govnr.TreeSupervisor
	network    *inmemory.Network
	clock      *adapter.AdjustableClock
	cancelFunc context.CancelFunc
	httpServer *httpserver.HttpServer
	logger     log.Logger
}

type ServerConfig struct {
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

func StartGammaServer(config ServerConfig) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	rootLogger := getLogger(config.Silent)

	clock := adapter.NewAdjustableClock()

	network, cfg := NewDevelopmentNetwork(ctx, rootLogger, clock, config)
	rootLogger.Info("finished creating development network")

	httpServer := httpserver.NewHttpServer(cfg,	rootLogger, network.MetricRegistry(0))
	httpServer.RegisterPublicApi(network.PublicApi(0))

	s := &Server{
		network:    network,
		clock:      clock,
		cancelFunc: cancel,
		httpServer: httpServer,
		logger:     rootLogger,
	}

	s.addGammaHandlers(httpServer.Router())

	s.Supervise(httpServer)
	s.Supervise(network)

	return s
}

func (s *Server) GracefulShutdown(shutdownContext context.Context) {
	s.cancelFunc()
	supervised.ShutdownAllGracefully(shutdownContext, s.httpServer)
}

var (
	port               = flag.Int("port", 8080, "The port to bind the gamma server to")
	profiling          = flag.Bool("profiling", false, "enable profiling")
	version            = flag.Bool("version", false, "returns information about version")
	overrideConfigJson = flag.String("override-config", "{}", "JSON-formatted config overrides, same format as the file config")
)

func Main(listening chan<- int) {

	flag.Parse()

	if *version {
		fmt.Println(config.GetVersion())
		return
	}

	requestedPort := strconv.Itoa(*port)
	if requestedPort == "-1" {
		requestedPort = "0"
	}
	var serverAddress = ":" + requestedPort

	gamma := StartGammaServer(ServerConfig{
		ServerAddress:      serverAddress,
		Profiling:          *profiling,
		OverrideConfigJson: *overrideConfigJson,
		Silent:             false,
	})

	supervised.NewShutdownListener(gamma.logger, gamma).ListenToOSShutdownSignal()
	if listening != nil {
		listening <- gamma.httpServer.Port()
	}
	gamma.WaitUntilShutdown(context.Background())

}
