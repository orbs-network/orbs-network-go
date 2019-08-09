// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package bootstrap

import (
	"context"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/scribe/log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type ShutdownWaiter interface {
	GracefulShutdown(shutdownContext context.Context)
	WaitUntilShutdown()
}

func ShutdownGracefully(waiter ShutdownWaiter, timeout time.Duration) {
	shutdownContext, cancel := context.WithTimeout(context.Background(), timeout) // give system some time to gracefully finish
	defer cancel()
	waiter.GracefulShutdown(shutdownContext)
}

func WaitForAllToShutdown(waiters ...ShutdownWaiter) {
	for _, w := range waiters {
		w.WaitUntilShutdown()
	}
}

func ShutdownAllGracefully(shutdownCtx context.Context, waiters ...ShutdownWaiter) {
	for _, w := range waiters {
		w.GracefulShutdown(shutdownCtx)
	}
}

type OSShutdownListener struct {
	Logger       log.Logger
	shutdownCond *sync.Cond
	waiter       ShutdownWaiter
}

func NewShutdownListener(logger log.Logger, waiter ShutdownWaiter) *OSShutdownListener {
	return &OSShutdownListener{
		shutdownCond: sync.NewCond(&sync.Mutex{}),
		Logger:       logger,
		waiter:     waiter,
	}
}

func (n *OSShutdownListener) ListenToOSShutdownSignal() {
	// if waiting for shutdown, listen for sigint and sigterm
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	supervised.GoOnce(n.Logger, func() {
		<-signalChan
		n.Logger.Info("terminating node gracefully due to os signal received")

		ShutdownGracefully(n.waiter, 100 * time.Millisecond)
	})

}


