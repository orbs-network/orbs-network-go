// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/govnr"
	"time"
)

func WithContext(f func(ctx context.Context)) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f(ctx)
}

func WithContextAndShutdown(waiter govnr.ShutdownWaiter, f func(ctx context.Context)) {
	ctx, cancel := context.WithCancel(context.Background())
	defer shutdown(waiter)
	defer cancel()
	f(ctx)
}

func shutdown(waiter govnr.ShutdownWaiter) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	waiter.WaitUntilShutdown(ctx)
}

func WithContextWithTimeout(d time.Duration, f func(ctx context.Context)) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	f(ctx)
}
