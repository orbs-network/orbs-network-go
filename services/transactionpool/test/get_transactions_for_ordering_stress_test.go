// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
	"time"
)

// gamma uses large time between empty blocks which is practically infinite and transactions are added slowly
func TestStress_GetTransactionsForOrderingWhenFirstTxAdded(t *testing.T) {
	const ITERATIONS = 1000
	for i := 0; i < ITERATIONS; i++ {

		test.WithContext(func(ctx context.Context) {
			h := newHarnessWithInfiniteTimeBetweenEmptyBlocks(t).start(ctx)
			h.ignoringForwardMessages()

			ch := make(chan int)

			go func() {
				time.Sleep(time.Duration(rand.Intn(1000)) * time.Microsecond)
				out, err := h.getTransactionsForOrdering(ctx, 2, 1)
				require.NoError(t, err)
				ch <- len(out.SignedTransactions)
			}()

			go func() {
				time.Sleep(time.Duration(rand.Intn(1000)) * time.Microsecond)
				tx := builders.TransferTransaction().Build()
				h.addNewTransaction(ctx, tx)
			}()

			numOfTxs := <-ch
			require.EqualValues(t, 1, numOfTxs, "did not the newly added transaction")
		})

	}
}
