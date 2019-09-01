// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package blockstorage

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestDetectForkReturnsErrorOnDifferentTimestamp(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		harness.AllowErrorsMatching("FORK!!")
		blockBuilder := builders.BlockPair()
		blockPair := blockBuilder.Build()

		mutatedBlockPair := blockBuilder.WithBlockCreated(time.Now().Add(1 * time.Hour)).Build()

		err := detectForks(mutatedBlockPair, blockPair.TransactionsBlock.Header, blockPair.ResultsBlock.Header, harness.Logger)

		require.EqualError(t, err, "FORK!! block already in storage, timestamp mismatch", "same block, different timestamp should return an error")
	})
}

func TestDetectForkReturnsErrorOnDifferentTxBlock(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		harness.AllowErrorsMatching("FORK!!")
		blockBuilder := builders.BlockPair()
		blockPair := blockBuilder.Build()

		mutatedBlockPair := blockBuilder.Build()
		mutatedBlockPair.TransactionsBlock.Header.MutateNumSignedTransactions(999)

		err := detectForks(mutatedBlockPair, blockPair.TransactionsBlock.Header, blockPair.ResultsBlock.Header, harness.Logger)

		require.EqualError(t, err, "FORK!! block already in storage, transaction block header mismatch", "same block, different transactions block header should return an error")
	})
}

func TestDetectForkReturnsErrorOnDifferentRxBlock(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		harness.AllowErrorsMatching("FORK!!")
		blockBuilder := builders.BlockPair()
		blockPair := blockBuilder.Build()
		mutatedBlockPair := blockBuilder.Build()
		mutatedBlockPair.ResultsBlock.Header.MutateNumTransactionReceipts(999)

		err := detectForks(mutatedBlockPair, blockPair.TransactionsBlock.Header, blockPair.ResultsBlock.Header, harness.Logger)

		require.EqualError(t, err, "FORK!! block already in storage, results block header mismatch", "same block, different results block header should return an error")
	})
}
