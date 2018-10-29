package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCommittedTransactionPoolClearsOldTransactions(t *testing.T) {
	t.Parallel()
	test.WithContext(func(ctx context.Context) {
		p := NewCommittedPool(metric.NewRegistry())

		r1 := builders.TransactionReceipt().WithRandomHash().Build()
		r2 := builders.TransactionReceipt().WithRandomHash().Build()
		r3 := builders.TransactionReceipt().WithRandomHash().Build()
		p.add(r1, primitives.TimestampNano(time.Now().Add(-5*time.Minute).UnixNano()))
		p.add(r2, primitives.TimestampNano(time.Now().Add(-29*time.Minute).UnixNano()))
		p.add(r3, primitives.TimestampNano(time.Now().Add(-31*time.Minute).UnixNano()))

		p.clearTransactionsOlderThan(ctx, time.Now().Add(-30*time.Minute))

		require.True(t, p.has(r1.Txhash()), "cleared non-expired transaction")
		require.True(t, p.has(r2.Txhash()), "cleared non-expired transaction")
		require.False(t, p.has(r3.Txhash()), "did not clear expired transaction")
	})
}
