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
}

func StartGammaServer(serverAddress string, blocking bool) *GammaServer {
	ctx, cancel := context.WithCancel(context.Background())

	network := harness.NewDevelopmentNetwork().StartNodes(ctx)
	testLogger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	metricRegistry := metric.NewRegistry()

	httpServer := httpserver.NewHttpServer(serverAddress, testLogger, network.PublicApi(0), metricRegistry)

	s := &GammaServer{
		ctxCancel:    cancel,
		shutdownCond: sync.NewCond(&sync.Mutex{}),
		httpServer:   httpServer,
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
