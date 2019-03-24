// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	. "github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/BenchmarkToken"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// Control group - if this fails, there are bugs unrelated to message tampering
func TestGazillionTxHappyFlow(t *testing.T) {
	rnd := rand.NewControlledRand(t)
	newHarness().
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			sendTransfersAndAssertTotalBalance(ctx, network, t, 200, rnd)
		})
}

func TestGazillionTxWhileDuplicatingMessages(t *testing.T) {
	rnd := rand.NewControlledRand(t)
	getStressTestHarness().
		AllowingErrors(
			"error adding forwarded transaction to pending pool", // because we duplicate, among other messages, the transaction propagation message
		).
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			network.TransportTamperer().Duplicate(WithPercentChance(rnd, 15))

			sendTransfersAndAssertTotalBalance(ctx, network, t, 100, rnd)
		})
}

// TODO (v1) Must drop message from up to "f" fixed nodes (for 4 nodes f=1)
func TestGazillionTxWhileDroppingMessages(t *testing.T) {
	rnd := rand.NewControlledRand(t)
	getStressTestHarness().
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			network.TransportTamperer().Fail(HasHeader(AConsensusMessage).And(WithPercentChance(rnd, 12)))

			sendTransfersAndAssertTotalBalance(ctx, network, t, 100, rnd)
		})
}

// See BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT - cannot delay messages consistently more than that, or block sync will never work - it throws "timed out when waiting for chunks"
func TestGazillionTxWhileDelayingMessages(t *testing.T) {
	rnd := rand.NewControlledRand(t)
	getStressTestHarness().
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			network.TransportTamperer().Delay(func() time.Duration {
				return (time.Duration(rnd.Intn(50))) * time.Millisecond
			}, WithPercentChance(rnd, 30))

			sendTransfersAndAssertTotalBalance(ctx, network, t, 100, rnd)
		})
}

// TODO (v1) Must corrupt message from up to "f" fixed nodes (for 4 nodes f=1)
func TestGazillionTxWhileCorruptingMessages(t *testing.T) {
	t.Skip("This should work - fix and remove Skip")
	rnd := rand.NewControlledRand(t)
	newHarness().
		AllowingErrors(
			"transport header is corrupt", // because we corrupt messages
		).
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			tamper := network.TransportTamperer().Corrupt(Not(HasHeader(ATransactionRelayMessage)).And(WithPercentChance(rnd, 15)), rnd)
			sendTransfersAndAssertTotalBalance(ctx, network, t, 90, rnd)
			tamper.StopTampering(ctx)

			sendTransfersAndAssertTotalBalance(ctx, network, t, 10, rnd)

		})
}

func WithPercentChance(ctrlRand *rand.ControlledRand, pct int) MessagePredicate {
	return func(data *adapter.TransportData) bool {
		if pct >= 100 {
			return true
		} else if pct <= 0 {
			return false
		} else {
			return ctrlRand.Intn(101) <= pct
		}
	}
}

func TestWithNPctChance_AlwaysTrue(t *testing.T) {
	ctrlRand := rand.NewControlledRand(t)
	require.True(t, WithPercentChance(ctrlRand, 100)(nil), "100% chance should always return true")
}

func TestWithNPctChance_AlwaysFalse(t *testing.T) {
	ctrlRand := rand.NewControlledRand(t)
	require.False(t, WithPercentChance(ctrlRand, 0)(nil), "0% chance should always return false")
}

func TestWithNPctChance_ManualCheck(t *testing.T) {
	ctrlRand := rand.NewControlledRand(t)
	tries := 1000
	pct := ctrlRand.Intn(100)
	hits := 0
	for i := 0; i < tries; i++ {
		if WithPercentChance(ctrlRand, pct)(nil) {
			hits++
		}
	}
	t.Logf("Manual test for WithPercentChance: Tries=%d Chance=%d%% Hits=%d\n", tries, pct, hits)
}

func sendTransfersAndAssertTotalBalance(ctx context.Context, network *NetworkHarness, t testing.TB, numTransactions int, ctrlRand *rand.ControlledRand) {
	fromAddress := 5
	toAddress := 6
	contract := network.DeployBenchmarkTokenContract(ctx, fromAddress)

	var expectedSum uint64 = 0
	var txHashes []primitives.Sha256
	for i := 0; i < numTransactions; i++ {
		amount := uint64(ctrlRand.Int63n(100))
		expectedSum += amount

		txHash := contract.TransferInBackground(ctx, ctrlRand.Intn(network.Size()), amount, fromAddress, toAddress)
		txHashes = append(txHashes, txHash)
	}
	for _, txHash := range txHashes {
		network.WaitForTransactionInState(ctx, txHash)
	}

	for i := 0; i < network.Size(); i++ {
		actualSum := contract.GetBalance(ctx, i, toAddress)
		require.EqualValuesf(t, expectedSum, actualSum, "recipient balance did not equal expected balance in node %d", i)

		actualRemainder := contract.GetBalance(ctx, i, fromAddress)
		require.EqualValuesf(t, benchmarktoken.TOTAL_SUPPLY-expectedSum, actualRemainder, "sender balance did not equal expected balance in node %d", i)
	}
}
