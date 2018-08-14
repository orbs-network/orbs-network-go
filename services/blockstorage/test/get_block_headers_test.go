package test

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReturnTransactionBlockHeader(t *testing.T) {
	driver := NewDriver(t)
	driver.expectCommitStateDiff()

	block := builders.BlockPair().Build()
	driver.commitBlock(block)

	output, err := driver.blockStorage.GetTransactionsBlockHeader(&services.GetTransactionsBlockHeaderInput{BlockHeight: 1})

	require.NoError(t, err)
	require.EqualValues(t, block.TransactionsBlock.Header, output.TransactionsBlockHeader)
	require.EqualValues(t, block.TransactionsBlock.Metadata, output.TransactionsBlockMetadata)
	require.EqualValues(t, block.TransactionsBlock.BlockProof, output.TransactionsBlockProof)
}

// FIXME time out
func TestReturnTransactionBlockHeaderFromNearFuture(t *testing.T) {
	driver := NewDriver(t)
	driver.expectCommitStateDiff()

	block := builders.BlockPair().Build()
	driver.commitBlock(block)

	result := make(chan *services.GetTransactionsBlockHeaderOutput)
	blockHeightInTheFuture := primitives.BlockHeight(5)

	go func() {
		output, _ := driver.blockStorage.GetTransactionsBlockHeader(&services.GetTransactionsBlockHeaderInput{BlockHeight: blockHeightInTheFuture})
		result <- output
	}()

	for i := primitives.BlockHeight(2); i <= blockHeightInTheFuture+1; i++ {
		driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(i)).Build())
	}

	require.EqualValues(t, blockHeightInTheFuture+1, driver.getLastBlockHeight().LastCommittedBlockHeight)

	output := <-result
	require.EqualValues(t, blockHeightInTheFuture, output.TransactionsBlockHeader.BlockHeight())
}

func TestReturnTransactionBlockHeaderFromNearFutureReturnsTimeout(t *testing.T) {
	driver := NewDriver(t)
	driver.expectCommitStateDiff()

	block := builders.BlockPair().Build()
	driver.commitBlock(block)

	timeoutError := make(chan error)
	blockHeightInTheFuture := primitives.BlockHeight(5)

	go func() {
		_, err := driver.blockStorage.GetTransactionsBlockHeader(&services.GetTransactionsBlockHeaderInput{BlockHeight: blockHeightInTheFuture})
		timeoutError <- err
	}()

	for i := primitives.BlockHeight(2); i <= 4; i++ {
		driver.commitBlock(builders.BlockPair().WithHeight(i).Build())
	}

	err := <-timeoutError
	require.EqualError(t, err, "operation timed out")
}

func TestReturnResultsBlockHeader(t *testing.T) {
	driver := NewDriver(t)
	driver.expectCommitStateDiff()

	block := builders.BlockPair().Build()
	driver.commitBlock(block)

	output, err := driver.blockStorage.GetResultsBlockHeader(&services.GetResultsBlockHeaderInput{BlockHeight: 1})

	require.NoError(t, err)
	require.EqualValues(t, block.ResultsBlock.Header, output.ResultsBlockHeader)
	require.EqualValues(t, block.ResultsBlock.BlockProof, output.ResultsBlockProof)
}

// FIXME time out
func TestReturnResultsBlockHeaderFromNearFuture(t *testing.T) {
	driver := NewDriver(t)
	driver.expectCommitStateDiff()

	block := builders.BlockPair().Build()
	driver.commitBlock(block)

	result := make(chan *services.GetResultsBlockHeaderOutput)
	blockHeightInTheFuture := primitives.BlockHeight(5)

	go func() {
		output, _ := driver.blockStorage.GetResultsBlockHeader(&services.GetResultsBlockHeaderInput{BlockHeight: blockHeightInTheFuture})
		result <- output
	}()

	for i := primitives.BlockHeight(2); i <= blockHeightInTheFuture+1; i++ {
		driver.commitBlock(builders.BlockPair().WithHeight(i).Build())
	}

	require.EqualValues(t, blockHeightInTheFuture+1, driver.getLastBlockHeight().LastCommittedBlockHeight)

	output := <-result

	require.EqualValues(t, blockHeightInTheFuture, output.ResultsBlockHeader.BlockHeight())
}

func TestReturnResultsBlockHeaderFromNearFutureReturnsTimeout(t *testing.T) {
	driver := NewDriver(t)
	driver.expectCommitStateDiff()

	block := builders.BlockPair().Build()
	driver.commitBlock(block)

	timeoutError := make(chan error)
	blockHeightInTheFuture := primitives.BlockHeight(5)

	go func() {
		_, err := driver.blockStorage.GetResultsBlockHeader(&services.GetResultsBlockHeaderInput{BlockHeight: blockHeightInTheFuture})
		timeoutError <- err
	}()

	for i := primitives.BlockHeight(2); i <= blockHeightInTheFuture-1; i++ {
		driver.commitBlock(builders.BlockPair().WithHeight(i).Build())
	}

	err := <-timeoutError
	require.EqualError(t, err, "operation timed out")
}
