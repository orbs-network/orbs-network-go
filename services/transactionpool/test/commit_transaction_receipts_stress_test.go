// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

func TestStress_AddingSameTransactionMultipleTimesWhileReportingAsCommitted(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		const CONCURRENCY_COUNT = 500
		duplicateStatuses := []protocol.TransactionStatus{
			protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED,
			protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING}
		startBarrier := sync.WaitGroup{}
		doneBarrier := sync.WaitGroup{}
		startBarrier.Add(CONCURRENCY_COUNT)
		doneBarrier.Add(CONCURRENCY_COUNT)

		h := newHarness(t).allowingErrorsMatching("error adding transaction to pending pool").start(ctx)
		h.ignoringForwardMessages()
		h.ignoringTransactionResults()
		h.ignoringBlockHeightChecks()

		outerCtx := trace.NewContext(ctx, "outer")
		tx := builders.TransferTransaction().Build()
		_, err := h.addNewTransaction(outerCtx, tx)
		require.NoError(t, err, "adding a transaction returned an unexpected error")

		for i := 0; i < CONCURRENCY_COUNT; i++ { // the idea here is to bombard addNewTransaction with a lot of concurrent calls for the same tx while simultaneously reporting tx as committed
			go func() {
				innerCtx := trace.NewContext(ctx, "inner")
				startBarrier.Done() // release one count, I'm ready for test
				startBarrier.Wait() // wait for the others

				receipt, _ := h.addNewTransaction(innerCtx, tx) // concurrently with h.reportTransactionsAsCommitted()

				assert.Contains(t, duplicateStatuses, receipt.TransactionStatus, "transaction must receive a duplicate status while in transit between pending and committed pools")
				doneBarrier.Done()
			}()
		}

		startBarrier.Wait() // wait until all goroutines are at the barrier so we can release all at the same time to tease out any flakiness

		_, err = h.reportTransactionsAsCommitted(outerCtx, tx) // concurrently with h.addNewTransaction()
		require.NoError(t, err, "committing a transaction returned an unexpected error")

		doneBarrier.Wait()
	})
}
