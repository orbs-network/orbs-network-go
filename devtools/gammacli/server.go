package gammacli

import (
	"context"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/bootstrap/httpserver"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"os"
	"sync"
	"time"
)

type GammaServer struct {
	httpServer   httpserver.HttpServer
	logic        bootstrap.NodeLogic
	shutdownCond *sync.Cond
	ctxCancel    context.CancelFunc
	Logger       log.BasicLogger
}

func StartGammaServer(serverAddress string, blocking bool) *GammaServer {
	ctx, cancel := context.WithCancel(context.Background())

	testLogger := log.GetLogger().
		WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter())).
		WithFilters(
			//TODO what do we really want to output to the gamma server log? maybe some meaningful data for our users?
			log.IgnoreMessagesMatching("Metric recorded"),
			log.IgnoreMessagesMatching("state transitioning"),
			log.IgnoreMessagesMatching("finished waiting for responses"),
			log.IgnoreMessagesMatching("no responses received"),
		)

	network := harness.NewDevelopmentNetwork(testLogger).StartNodes(ctx)
	testLogger.Info("finished creating development network")

	metricRegistry := metric.NewRegistry()

	httpServer := httpserver.NewHttpServer(serverAddress, testLogger, network.PublicApi(0), metricRegistry)

	s := &GammaServer{
		ctxCancel:    cancel,
		shutdownCond: sync.NewCond(&sync.Mutex{}),
		httpServer:   httpServer,
		Logger: testLogger,
	}

	if blocking == true {
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
