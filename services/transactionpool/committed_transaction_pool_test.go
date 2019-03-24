// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCommittedTransactionPoolClearsOldTransactions(t *testing.T) {
	t.Parallel()

	test.WithContext(func(ctx context.Context) {
		p := NewCommittedPool(func() time.Duration { return time.Second }, metric.NewRegistry())
		ctrlRand := rand.NewControlledRand(t)

		r1 := builders.TransactionReceipt().WithRandomHash(ctrlRand).Build()
		r2 := builders.TransactionReceipt().WithRandomHash(ctrlRand).Build()
		r3 := builders.TransactionReceipt().WithRandomHash(ctrlRand).Build()

		p.add(r1, primitives.BlockHeight(3), primitives.TimestampNano(time.Now().Add(-5*time.Minute).UnixNano()))
		p.add(r2, primitives.BlockHeight(2), primitives.TimestampNano(time.Now().Add(-29*time.Minute).UnixNano()))
		p.add(r3, primitives.BlockHeight(1), primitives.TimestampNano(time.Now().Add(-31*time.Minute).UnixNano()))

		p.clearTransactionsOlderThan(ctx, primitives.TimestampNano(time.Now().Add(-30*time.Minute).UnixNano()))

		require.True(t, p.has(r1.Txhash()), "cleared non-expired transaction")
		require.True(t, p.has(r2.Txhash()), "cleared non-expired transaction")
		require.False(t, p.has(r3.Txhash()), "did not clear expired transaction")
	})
}
