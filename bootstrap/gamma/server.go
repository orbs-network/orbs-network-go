// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package gamma

import (
	"context"
	"github.com/orbs-network/orbs-network-go/bootstrap/httpserver"
	"github.com/orbs-network/scribe/log"
	"os"
	"sync"
	"time"
)

type GammaServer struct {
	httpServer   httpserver.HttpServer
	shutdownCond *sync.Cond
	ctxCancel    context.CancelFunc
	Logger       log.Logger
}

func StartGammaServer(serverAddress string, profiling bool, overrideConfigJson string, blocking bool) *GammaServer {
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
		ctxCancel:    cancel,
		shutdownCond: sync.NewCond(&sync.Mutex{}),
		httpServer:   httpServer,
		Logger:       rootLogger,
	}

	if blocking {
		s.WaitUntilShutdown()
	} else { // Used primarily in testing
		go s.WaitUntilShutdown()
	}

	return s
}

func (n *GammaServer) GracefulShutdown(timeout time.Duration) {
	n.ctxCancel()
	n.httpServer.GracefulShutdown(timeout)
	n.shutdownCond.Broadcast()
}

func (n *GammaServer) WaitUntilShutdown() {
	n.shutdownCond.L.Lock()
	n.shutdownCond.Wait()
	n.shutdownCond.L.Unlock()
}

func (n *GammaServer) Port() int {
	return n.httpServer.Port()
}
