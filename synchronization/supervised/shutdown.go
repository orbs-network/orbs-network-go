// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package supervised

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/scribe/log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type GracefulShutdowner interface {
	govnr.ShutdownWaiter
	GracefulShutdown(shutdownContext context.Context)
}

type ChanShutdownWaiter struct {
	closed      chan struct{}
	description string
}

func (c *ChanShutdownWaiter) WaitUntilShutdown(shutdownContext context.Context) {
	select {
	case <-c.closed:
	case <-shutdownContext.Done():
		if shutdownContext.Err() == context.DeadlineExceeded {
			panic(fmt.Sprintf("failed to shutdown %s before timeout", c.description))
		}
	}
}

func (c *ChanShutdownWaiter) Shutdown() {
	close(c.closed)
}

func NewChanWaiter(description string) ChanShutdownWaiter {
	return ChanShutdownWaiter{closed: make(chan struct{}), description: description}
}

func ShutdownGracefully(s GracefulShutdowner, timeout time.Duration) {
	shutdownContext, cancel := context.WithTimeout(context.Background(), timeout) // give system some time to gracefully finish
	defer cancel()
	s.GracefulShutdown(shutdownContext)
}

func ShutdownAllGracefully(shutdownCtx context.Context, shutdowners ...GracefulShutdowner) {
	for _, w := range shutdowners {
		w.GracefulShutdown(shutdownCtx)
	}
}

type OSShutdownListener struct {
	Logger       log.Logger
	shutdownCond *sync.Cond
	shutdowner   GracefulShutdowner
}

func NewShutdownListener(logger log.Logger, shutdowner GracefulShutdowner) *OSShutdownListener {
	return &OSShutdownListener{
		shutdownCond: sync.NewCond(&sync.Mutex{}),
		Logger:       logger,
		shutdowner:   shutdowner,
	}
}

func (n *OSShutdownListener) ListenToOSShutdownSignal() {
	// if waiting for shutdown, listen for sigint and sigterm
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	govnr.Once(logfields.GovnrErrorer(n.Logger), func() {
		<-signalChan
		n.Logger.Info("terminating node gracefully due to os signal received")

		ShutdownGracefully(n.shutdowner, 100*time.Millisecond)
	})

}
