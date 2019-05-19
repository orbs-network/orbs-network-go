package bootstrap

import (
	"context"
	"github.com/orbs-network/orbs-network-go/bootstrap/httpserver"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/scribe/log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type OrbsProcess struct {
	CancelFunc   context.CancelFunc
	Logger       log.Logger
	HttpServer   httpserver.HttpServer
	shutdownCond *sync.Cond
}

func NewOrbsProcess(logger log.Logger, cancelFunc context.CancelFunc, httpserver httpserver.HttpServer) OrbsProcess {
	return OrbsProcess{
		shutdownCond: sync.NewCond(&sync.Mutex{}),
		Logger:       logger,
		CancelFunc:   cancelFunc,
		HttpServer:   httpserver,
	}
}

func (n *OrbsProcess) GracefulShutdown(timeout time.Duration) {
	n.CancelFunc()
	n.HttpServer.GracefulShutdown(timeout)
	n.shutdownCond.Broadcast()
}

func (n *OrbsProcess) WaitUntilShutdown() {
	// if waiting for shutdown, listen for sigint and sigterm
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	supervised.GoOnce(n.Logger, func() {
		<-signalChan
		n.Logger.Info("terminating node gracefully due to os signal received")
		n.GracefulShutdown(0)
	})

	n.shutdownCond.L.Lock()
	n.shutdownCond.Wait()
	n.shutdownCond.L.Unlock()
}
