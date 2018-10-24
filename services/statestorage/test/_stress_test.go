package test

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

const TpxAvg = 100
const TxCount6Months100Tps int = 6 * 30 * 24 * 60 * 60 * TpxAvg
const BlockSizeMax = 200
const Users = 1000000


func TestSimulateStateInitFlowForSixMonthsAt100Tps(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		d := newStateStorageDriver(1)

		// generate User keys
		userKeys := make([][]byte, Users)
		for i := range userKeys {
			userKeys[i] = make([]byte, 32)
			rand.Read(userKeys[i])
		}

		var txCount int
		var commits int
		var createInputDuration time.Duration
		var commitDuration time.Duration
		start := time.Now()
		tickStart := start
		for txCount < TxCount6Months100Tps {
			// create input for current simulated block
			generationStart := time.Now()
			commitTxs, commit := getRandomCommit(userKeys)

			createInputDuration += time.Now().Sub(generationStart)
			txCount += commitTxs

			// commit state
			d.commitStateDiff(ctx, commit)

			// print progress every 100000 commits
			if commits++; commits % 100000 == 0 {
				ms := runtime.MemStats{}
				runtime.ReadMemStats(&ms)
				elapsedTick := time.Now().Sub(tickStart)
				elapsed := time.Now().Sub(start)
				commitDuration = elapsed - createInputDuration
				fmt.Printf("tick: %v, commit: %v, elapsed: %v, progress: %d, HeapSys: %dMB, HeapAlloc: %dMB\n", elapsedTick, commitDuration, elapsed, 100*txCount/TxCount6Months100Tps, ms.HeapSys/ (1024 * 1024), ms.HeapAlloc/ (1024 * 1024))
				tickStart = time.Now()
			}
		}

		// print summary
		commitDuration = time.Now().Sub(start) - createInputDuration
		fmt.Printf("Wrote    %v transactions in %v blocks to state for %v users in %v\f", txCount, commits, Users, time.Now().Sub(start))

		require.WithinDuration(t, start, start.Add(commitDuration), time.Minute)
	})
}

func getRandomCommit(userKeys [][]byte) (int, *services.CommitStateDiffInput) {
	blockSize := rand.Int() % BlockSizeMax
	blockDiff := builders.ContractStateDiff().WithContractName("someContract")
	for i := 0; i < blockSize; i++ {
		address := userKeys[rand.Int()%len(userKeys)]
		blockDiff.WithStringRecord(string(address), string(rand.Uint64()))
	}
	commit := CommitStateDiff().WithBlockHeight(1).WithDiff(blockDiff.Build()).Build()
	return blockSize, commit
}
