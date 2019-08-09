// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package supervised

import (
	"context"
	"github.com/orbs-network/scribe/log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type ShutdownWaiter interface {
	WaitUntilShutdown()
}

type GracefulShutdowner interface {
	ShutdownWaiter
	GracefulShutdown(shutdownContext context.Context)
}

type ChanWaiter struct {
	Closed chan struct{}
}

func (c *ChanWaiter) WaitUntilShutdown() {
	<-c.Closed
}

func (c *ChanWaiter) Shutdown() {
	close(c.Closed)
}

func chanWaiterFor(ch chan struct{}) *ChanWaiter {
	return &ChanWaiter{Closed: ch}
}

func NewChanWaiter() ChanWaiter {
	return ChanWaiter{Closed: make(chan struct{})}
}

type TreeSupervisor struct {
	supervised []ShutdownWaiter
}

func (t *TreeSupervisor) WaitUntilShutdown() {
	for _, w := range t.supervised {
		w.WaitUntilShutdown()
	}
}

func (t *TreeSupervisor) Supervise(w ShutdownWaiter) {
	t.supervised = append(t.supervised, w)
}

func (t *TreeSupervisor) SuperviseChan(ch chan struct{}) {
	t.supervised = append(t.supervised, chanWaiterFor(ch))
}

func ShutdownGracefully(s GracefulShutdowner, timeout time.Duration) {
	shutdownContext, cancel := context.WithTimeout(context.Background(), timeout) // give system some time to gracefully finish
	defer cancel()
	s.GracefulShutdown(shutdownContext)
}

func WaitForAllToShutdown(waiters ...ShutdownWaiter) {
	for _, w := range waiters {
		w.WaitUntilShutdown()
	}
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
