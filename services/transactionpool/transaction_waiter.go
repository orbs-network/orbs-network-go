// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package transactionpool

import "context"

//Kind of a barrier which waits until a specific number of notifications have been met, or until a context is done
//Note: not thread-safe; do not reuse the same instance in two goroutines
type transactionWaiter struct {
	incremented chan struct{}
}

func (w *transactionWaiter) waitForIncomingTransaction(ctx context.Context) bool {
	select {
	case <-w.incremented:
		return true
	case <-ctx.Done():
		return false
	}
}

func (w *transactionWaiter) inc(ctx context.Context) {
	select {
	case w.incremented <- struct{}{}:
	default:
		return
	}
}

func newTransactionWaiter() *transactionWaiter {
	return &transactionWaiter{incremented: make(chan struct{}, 1)}
}
