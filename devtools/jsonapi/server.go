package jsonapi

import (
	"context"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/bootstrap/httpserver"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"os"
	"sync"
	"time"
)

var testLogger = log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

type Sambusac struct {
	httpServer   httpserver.HttpServer
	logic        bootstrap.NodeLogic
	shutdownCond *sync.Cond
	ctxCancel    context.CancelFunc
}

func StartSambusac(serverAddress string, pathToContracts string, blocking bool) *Sambusac {
	ctx, cancel := context.WithCancel(context.Background())

	network := harness.NewDevelopmentNetwork().StartNodes(ctx)

	httpServer := httpserver.NewHttpServer(serverAddress, testLogger, network.PublicApi(0))

	s := &Sambusac{
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

func (n *Sambusac) GracefulShutdown(timeout time.Duration) {
	n.ctxCancel()
	n.httpServer.GracefulShutdown(timeout)
	n.shutdownCond.Broadcast()
}

func (n *Sambusac) WaitUntilShutdown() {
	n.shutdownCond.L.Lock()
	n.shutdownCond.Wait()
	n.shutdownCond.L.Unlock()
}
