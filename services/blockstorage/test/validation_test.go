package test

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidateBlockWithValidProtocolVersion(t *testing.T) {
	driver := NewDriver()
	block := builders.BlockPair().Build()

	_, err := driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
	require.NoError(t, err)
}

func TestValidateBlockWithInvalidProtocolVersion(t *testing.T) {
	driver := NewDriver()
	driver.t = t
	block := builders.BlockPair().Build()

	block.TransactionsBlock.Header.MutateProtocolVersion(998)

	_, err := driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
	require.EqualError(t, err, "protocol version mismatch")

	block.ResultsBlock.Header.MutateProtocolVersion(999)

	_, err = driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
	require.EqualError(t, err, "protocol version mismatch")

	block.TransactionsBlock.Header.MutateProtocolVersion(999)

	_, err = driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
	require.EqualError(t, err, "protocol version mismatch")

	block.TransactionsBlock.Header.MutateProtocolVersion(1)

	_, err = driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
	require.EqualError(t, err, "protocol version mismatch")
}

func TestValidateBlockWithValidHeight(t *testing.T) {
	driver := NewDriver()
	driver.t = t
	driver.expectCommitStateDiff()

	driver.commitBlock(builders.BlockPair().Build())

	block := builders.BlockPair().WithHeight(2).Build()

	_, err := driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
	require.NoError(t, err)
}

func TestValidateBlockWithInvalidHeight(t *testing.T) {
	driver := NewDriver()
	driver.t = t
	driver.expectCommitStateDiff()

	driver.commitBlock(builders.BlockPair().Build())

	block := builders.BlockPair().WithHeight(2).Build()

	block.TransactionsBlock.Header.MutateBlockHeight(998)

	_, err := driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
	require.EqualError(t, err, "block height is 998, expected 2")

	block.ResultsBlock.Header.MutateBlockHeight(999)

	_, err = driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
	require.EqualError(t, err, "block height is 998, expected 2")

	block.TransactionsBlock.Header.MutateBlockHeight(999)

	_, err = driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
	require.EqualError(t, err, "block height is 999, expected 2")

	block.TransactionsBlock.Header.MutateProtocolVersion(1)

	_, err = driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
	require.EqualError(t, err, "block height is 999, expected 2")
}

//TODO validate virtual chain
//TODO validate transactions root hash
//TODO validate metadata hash
//TODO validate receipts root hash
//TODO validate state diff hash
//TODO validate block consensus
