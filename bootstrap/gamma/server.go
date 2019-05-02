// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package gamma

import (
	"context"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/bootstrap/httpserver"
	"github.com/orbs-network/orbs-network-go/bootstrap/inmemory"
	"github.com/orbs-network/scribe/log"
	"os"
)

type GammaServer struct {
	bootstrap.OrbsProcess
	network *inmemory.Network
}

func StartGammaServer(serverAddress string, profiling bool, overrideConfigJson string) *GammaServer {
	ctx, cancel := context.WithCancel(context.Background())

	rootLogger := log.GetLogger().
		WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter())).
		WithFilters(
			//TODO(https://github.com/orbs-network/orbs-network-go/issues/585) what do we really want to output to the gamma server log? maybe some meaningful data for our users?
			log.IgnoreMessagesMatching("state transitioning"),
			log.IgnoreMessagesMatching("finished waiting for responses"),
			log.IgnoreMessagesMatching("no responses received"),
		)

	network := NewDevelopmentNetwork(ctx, rootLogger, overrideConfigJson)
	rootLogger.Info("finished creating development network")

	httpServer := httpserver.NewHttpServer(httpserver.NewServerConfig(serverAddress, profiling),
		rootLogger, network.PublicApi(0), network.MetricRegistry(0))

	s := &GammaServer{
		OrbsProcess: bootstrap.NewOrbsProcess(rootLogger, cancel, httpServer),
		network:     network,
	}

	return s
}

func (n *GammaServer) Port() int {
	return n.HttpServer.Port()
}
