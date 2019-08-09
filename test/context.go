// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"time"
)

func WithContext(f func(ctx context.Context)) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f(ctx)
}

func WithContextAndShutdown(waiter supervised.ShutdownWaiter, f func(ctx context.Context)) {
	ctx, cancel := context.WithCancel(context.Background())
	defer waiter.WaitUntilShutdown()
	defer cancel()
	f(ctx)
}

func WithContextWithTimeout(d time.Duration, f func(ctx context.Context)) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	f(ctx)
}
