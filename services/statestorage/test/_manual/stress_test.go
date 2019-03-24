// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package _manual

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/merkle"
	. "github.com/orbs-network/orbs-network-go/services/statestorage/test"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"math"
	"runtime"
	"testing"
	"time"
)

const TX_AVG_TPS = 100
const TX_COUNT_SIX_MONTHS_AT_AVG_TPS int = 6 * 30 * 24 * 60 * 60 * TX_AVG_TPS
const MAX_BLOCK_SIZE = 200
const USERS = 1000000

func TestSimulateMerkleInitForAllUsers(t *testing.T) {
	ctrlRand := rand.NewControlledRand(t)
	start := time.Now()

	userKeys := randomUsers(ctrlRand)

	ms := runtime.MemStats{}
	runtime.ReadMemStats(&ms)
	t.Logf("Finished init phase in %v. HeapAlloc is %dMB", time.Now().Sub(start), ms.HeapAlloc/(1024*1024))

	start = time.Now()
	diffs := make(merkle.TrieDiffs, 0, len(userKeys))
	for _, u := range userKeys {
		sha256 := hash.CalcSha256(u)
		diffs = append(diffs, &merkle.TrieDiff{
			Key:   u,
			Value: sha256,
		})
	}

	forest, root := merkle.NewForest()
	newRoot, err := forest.Update(root, diffs)
	require.NoError(t, err)
	require.NotEqual(t, root, newRoot)
	duration := time.Now().Sub(start)
	runtime.GC()
	runtime.GC()
	runtime.GC()
	runtime.GC()
	ms = runtime.MemStats{}
	runtime.ReadMemStats(&ms)
	t.Logf("Finished merkle build phase (%v keys) in %v. HeapAlloc is %dMB", len(userKeys), duration, ms.HeapAlloc/(1024*1024))

	user500Proof, err := forest.GetProof(newRoot, userKeys[500])
	t.Logf("user 500 value proof length is %v", len(user500Proof))
	require.NoError(t, err)

	valid, err := forest.Verify(newRoot, user500Proof, userKeys[500], hash.CalcSha256(userKeys[500]))
	require.True(t, valid)
	require.NoError(t, err)

	require.WithinDuration(t, start, time.Now(), 30*time.Second, "Expected Merkle to be populated in 30 seconds")
	require.True(t, ms.HeapAlloc < 500*1024*1024, "Expected memory use to be below 0.5GB")
}

func TestSimulateStateInitFlowForSixMonthsAt100Tps(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		ctrlRand := rand.NewControlledRand(t)

		d := NewStateStorageDriver(1)

		// generate User keys
		userKeys := randomUsers(ctrlRand)

		keysWritten := make(map[string]bool)
		start := time.Now()
		txCount, blockCount, commitDuration := loadTransactions(userKeys, keysWritten, d, ctx, t, ctrlRand)

		// print summary
		t.Logf("Wrote    %v transactions in %v blocks to state for %v users in %v\f", txCount, blockCount, USERS, time.Now().Sub(start))

		require.WithinDuration(t, start, start.Add(commitDuration), 10*time.Minute)

		// test state volume
		require.True(t, len(keysWritten) >= int(math.Min(float64(TX_COUNT_SIX_MONTHS_AT_AVG_TPS/4), USERS)))
		h, _, _ := d.GetBlockHeightAndTimestamp(ctx)
		for userId := range keysWritten { // verify all state entries were recorded
			value, _ := d.ReadSingleKeyFromRevision(ctx, h, "someContract", userId)
			require.NotZero(t, len(value))
		}

		// test memory consumption
		runtime.GC()
		runtime.GC()
		runtime.GC()
		runtime.GC()
		ms := runtime.MemStats{}
		runtime.ReadMemStats(&ms)
		require.True(t, ms.HeapAlloc < 2*1024*1024*1024) // using less than 2GB
	})
}

func randomUsers(ctrlRand *test.ControlledRand) [][]byte {
	userKeys := make([][]byte, USERS)
	for i := range userKeys {
		userKeys[i] = make([]byte, 32)
		ctrlRand.Read(userKeys[i])
	}
	return userKeys
}

func loadTransactions(userKeys [][]byte, keysWritten map[string]bool, d *Driver, ctx context.Context, t *testing.T, ctrlRand *test.ControlledRand) (int, int, time.Duration) {
	var txCount int
	var blockCount int
	var generatingInputDuration time.Duration
	var commitDuration time.Duration
	start := time.Now()
	tickStart := start
	var nextBlockHeight primitives.BlockHeight = 1
	for txCount < TX_COUNT_SIX_MONTHS_AT_AVG_TPS { // create input for current simulated block
		generationStart := time.Now()
		commitTxs, commit := generateRandomBlockStateDiff(userKeys, keysWritten, nextBlockHeight, "someContract", ctrlRand)

		generatingInputDuration += time.Now().Sub(generationStart)
		txCount += commitTxs

		// commit state
		output, err := d.CommitStateDiff(ctx, commit)
		require.NoError(t, err)
		nextBlockHeight = output.NextDesiredBlockHeight

		// print progress
		if blockCount++; blockCount%100 == 0 {
			ms := runtime.MemStats{}
			runtime.ReadMemStats(&ms)
			elapsedTick := time.Now().Sub(tickStart)
			elapsed := time.Now().Sub(start)
			commitDuration = elapsed - generatingInputDuration
			fmt.Printf("delta: %v, elapsed committing: %v, elapsed: %v, progress: %d, HeapSys: %dMB, HeapAlloc: %dMB, tx: %d, entries: %d\n",
				elapsedTick,
				commitDuration,
				elapsed,
				100*txCount/TX_COUNT_SIX_MONTHS_AT_AVG_TPS,
				ms.HeapSys/(1024*1024),
				ms.HeapAlloc/(1024*1024),
				txCount,
				len(keysWritten))
			tickStart = time.Now()
		}
	}
	commitDuration = time.Now().Sub(start) - generatingInputDuration
	return txCount, blockCount, commitDuration
}

func generateRandomBlockStateDiff(userKeys [][]byte, keysWritten map[string]bool, height primitives.BlockHeight, contract string, ctrlRand *test.ControlledRand) (int, *services.CommitStateDiffInput) {
	blockSize := ctrlRand.Int() % MAX_BLOCK_SIZE
	blockDiff := builders.ContractStateDiff().WithContractName(contract)
	for i := 0; i < blockSize; i++ {
		randUser := userKeys[ctrlRand.Int()%len(userKeys)]
		keysWritten[string(randUser)] = true
		randBalance := fmt.Sprintf("%x"+"", ctrlRand.Uint64())
		blockDiff.WithStringRecord(string(randUser), randBalance)
	}
	commit := CommitStateDiff().WithBlockHeight(int(height)).WithDiff(blockDiff.Build()).Build()
	return blockSize, commit
}
