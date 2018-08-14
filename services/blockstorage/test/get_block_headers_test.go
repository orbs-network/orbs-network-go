package test

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReturnTransactionBlockHeader(t *testing.T) {
	driver := NewDriver()
	driver.t = t
	driver.expectCommitStateDiff()

	block := builders.BlockPair().Build()
	driver.commitBlock(block)

	output, err := driver.blockStorage.GetTransactionsBlockHeader(&services.GetTransactionsBlockHeaderInput{BlockHeight: 1})

	require.NoError(t, err)
	require.EqualValues(t, output.TransactionsBlockHeader, block.TransactionsBlock.Header)
	require.EqualValues(t, output.TransactionsBlockMetadata, block.TransactionsBlock.Metadata)
	require.EqualValues(t, output.TransactionsBlockProof, block.TransactionsBlock.BlockProof)
}

// FIXME time out
func TestReturnTransactionBlockHeaderFromNearFuture(t *testing.T) {
	driver := NewDriver()
	driver.t = t
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

	require.EqualValues(t, driver.getLastBlockHeight().LastCommittedBlockHeight, blockHeightInTheFuture+1)

	output := <-result
	require.EqualValues(t, output.TransactionsBlockHeader.BlockHeight(), blockHeightInTheFuture)
}

func TestReturnTransactionBlockHeaderFromNearFutureReturnsTimeout(t *testing.T) {
	driver := NewDriver()
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
	require.Error(t, err, "operation timed out")
}

func TestReturnResultsBlockHeader(t *testing.T) {
	driver := NewDriver()
	driver.t = t
	driver.expectCommitStateDiff()

	block := builders.BlockPair().Build()
	driver.commitBlock(block)

	output, err := driver.blockStorage.GetResultsBlockHeader(&services.GetResultsBlockHeaderInput{BlockHeight: 1})

	require.NoError(t, err)
	require.EqualValues(t, output.ResultsBlockHeader, block.ResultsBlock.Header)
	require.EqualValues(t, output.ResultsBlockProof, block.ResultsBlock.BlockProof)
}

// FIXME time out
func TestReturnResultsBlockHeaderFromNearFuture(t *testing.T) {
	driver := NewDriver()
	driver.t = t
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

	require.EqualValues(t, driver.getLastBlockHeight().LastCommittedBlockHeight, blockHeightInTheFuture+1)

	output := <-result

	require.EqualValues(t, output.ResultsBlockHeader.BlockHeight(), blockHeightInTheFuture)
}

func TestReturnResultsBlockHeaderFromNearFutureReturnsTimeout(t *testing.T) {
	driver := NewDriver()
	driver.t = t
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
	require.Error(t, err, "operation timed out")
}
