package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestReturnTransactionBlockHeader(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness().
			withSyncBroadcast(1).
			withCommitStateDiff(1).
			withValidateConsensusAlgos(1).
			start(ctx)

		block := builders.BlockPair().Build()
		harness.commitBlock(ctx, block)

		output, err := harness.blockStorage.GetTransactionsBlockHeader(ctx, &services.GetTransactionsBlockHeaderInput{BlockHeight: 1})

		require.NoError(t, err, "this is a happy flow test")
		require.EqualValues(t, block.TransactionsBlock.Header, output.TransactionsBlockHeader, "block header data should be as committed")
		require.EqualValues(t, block.TransactionsBlock.Metadata, output.TransactionsBlockMetadata, "block header data should be as committed")
		require.EqualValues(t, block.TransactionsBlock.BlockProof, output.TransactionsBlockProof, "block header data should be as committed")
	})
}

func TestReturnTransactionBlockHeaderFromNearFuture(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness().
			withSyncBroadcast(1).
			withCommitStateDiff(1).
			withValidateConsensusAlgos(1).
			start(ctx)

		block := builders.BlockPair().Build()
		harness.commitBlock(ctx, block)

		result := make(chan *services.GetTransactionsBlockHeaderOutput)
		blockHeightInTheFuture := primitives.BlockHeight(5)

		go func() {
			output, _ := harness.blockStorage.GetTransactionsBlockHeader(ctx, &services.GetTransactionsBlockHeaderInput{BlockHeight: blockHeightInTheFuture})
			result <- output
		}()

		for i := primitives.BlockHeight(2); i <= blockHeightInTheFuture+1; i++ {
			harness.commitBlock(ctx, builders.BlockPair().WithHeight(primitives.BlockHeight(i)).Build())
		}

		require.EqualValues(t, blockHeightInTheFuture+1, harness.getLastBlockHeight(ctx, t).LastCommittedBlockHeight, "verify the test executed fully")

		output := <-result
		require.EqualValues(t, blockHeightInTheFuture, output.TransactionsBlockHeader.BlockHeight(), "block height should be 'in the future'")
	})
}

func TestReturnTransactionBlockHeaderFromNearFutureReturnsTimeout(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness().
			withBlockTrackerTimeout(30 * time.Millisecond).
			withSyncBroadcast(1).
			withCommitStateDiff(1).
			withValidateConsensusAlgos(1).
			start(ctx)

		block := builders.BlockPair().Build()
		harness.commitBlock(ctx, block)

		timeoutError := make(chan error)
		blockHeightInTheFuture := primitives.BlockHeight(5)

		go func() {
			_, err := harness.blockStorage.GetTransactionsBlockHeader(ctx, &services.GetTransactionsBlockHeaderInput{BlockHeight: blockHeightInTheFuture})
			timeoutError <- err
		}()

		for i := primitives.BlockHeight(2); i <= 4; i++ {
			harness.commitBlock(ctx, builders.BlockPair().WithHeight(i).Build())
		}

		err := <-timeoutError
		require.EqualError(t, err, "timed out waiting for block at height 5", "expect a timeout as the requested block height never reached")
	})
}

func TestReturnResultsBlockHeader(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness().
			withSyncBroadcast(1).
			withCommitStateDiff(1).
			withValidateConsensusAlgos(1).
			start(ctx)

		block := builders.BlockPair().Build()
		harness.commitBlock(ctx, block)

		output, err := harness.blockStorage.GetResultsBlockHeader(ctx, &services.GetResultsBlockHeaderInput{BlockHeight: 1})

		require.NoError(t, err, "results block happy flow")
		require.EqualValues(t, block.ResultsBlock.Header, output.ResultsBlockHeader, "block header data should be as committed")
		require.EqualValues(t, block.ResultsBlock.BlockProof, output.ResultsBlockProof, "block header data should be as committed")
	})
}

func TestReturnResultsBlockHeaderFromNearFuture(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness().
			withSyncBroadcast(1).
			withCommitStateDiff(1).
			withValidateConsensusAlgos(1).
			start(ctx)

		block := builders.BlockPair().Build()
		harness.commitBlock(ctx, block)

		result := make(chan *services.GetResultsBlockHeaderOutput)
		blockHeightInTheFuture := primitives.BlockHeight(5)

		go func() {
			output, _ := harness.blockStorage.GetResultsBlockHeader(ctx, &services.GetResultsBlockHeaderInput{BlockHeight: blockHeightInTheFuture})
			result <- output
		}()

		for i := primitives.BlockHeight(2); i <= blockHeightInTheFuture+1; i++ {
			harness.commitBlock(ctx, builders.BlockPair().WithHeight(i).Build())
		}

		require.EqualValues(t, blockHeightInTheFuture+1, harness.getLastBlockHeight(ctx, t).LastCommittedBlockHeight, "verify the test executed fully")

		output := <-result

		require.EqualValues(t, blockHeightInTheFuture, output.ResultsBlockHeader.BlockHeight(), "block height should be 'in the future'")
	})
}

func TestReturnResultsBlockHeaderFromNearFutureReturnsTimeout(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness().
			withBlockTrackerTimeout(30 * time.Millisecond).
			withSyncBroadcast(1).
			withCommitStateDiff(1).
			withValidateConsensusAlgos(1).
			start(ctx)

		block := builders.BlockPair().Build()
		harness.commitBlock(ctx, block)

		timeoutError := make(chan error)
		blockHeightInTheFuture := primitives.BlockHeight(5)

		go func() {
			_, err := harness.blockStorage.GetResultsBlockHeader(ctx, &services.GetResultsBlockHeaderInput{BlockHeight: blockHeightInTheFuture})
			timeoutError <- err
		}()

		for i := primitives.BlockHeight(2); i <= blockHeightInTheFuture-1; i++ {
			harness.commitBlock(ctx, builders.BlockPair().WithHeight(i).Build())
		}

		err := <-timeoutError
		require.EqualError(t, err, "timed out waiting for block at height 5", "expect a timeout as the requested block height never reached")
	})
}
