// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package supervised

import (
	"context"
	"fmt"
	"github.com/orbs-network/scribe/log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type ShutdownWaiter interface {
	WaitUntilShutdown(shutdownContext context.Context)
}

type GracefulShutdowner interface {
	ShutdownWaiter
	GracefulShutdown(shutdownContext context.Context)
}

type ChanWaiter struct {
	Closed      chan struct{}
	description string
}

func (c *ChanWaiter) WaitUntilShutdown(shutdownContext context.Context) {
	select {
	case <-c.Closed:
	case <-shutdownContext.Done():
		panic(fmt.Sprintf("failed to shutdown %s before timeout", c.description))
	}

}

func (c *ChanWaiter) Shutdown() {
	close(c.Closed)
}

func NewChanWaiter(description string) ChanWaiter {
	return ChanWaiter{Closed: make(chan struct{}), description: description}
}

type TreeSupervisor struct {
	supervised []ShutdownWaiter
}

func (t *TreeSupervisor) WaitUntilShutdown(shutdownContext context.Context) {
	for _, w := range t.supervised {
		w.WaitUntilShutdown(shutdownContext)
	}
}

func (t *TreeSupervisor) Supervise(w ShutdownWaiter) {
	t.supervised = append(t.supervised, w)
}

func (t *TreeSupervisor) SuperviseChan(description string, ch chan struct{}) {
	t.supervised = append(t.supervised, &ChanWaiter{Closed: ch, description: description})
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
	GoOnce(n.Logger, func() {
		<-signalChan
		n.Logger.Info("terminating node gracefully due to os signal received")

		ShutdownGracefully(n.shutdowner, 100*time.Millisecond)
	})

}
