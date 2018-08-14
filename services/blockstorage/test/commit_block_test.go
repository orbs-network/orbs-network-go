package test

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCommitBlockSavesToPersistentStorage(t *testing.T) {
	driver := NewDriver()
	driver.t = t

	driver.expectCommitStateDiff()

	blockCreated := time.Now()
	blockHeight := primitives.BlockHeight(1)

	_, err := driver.commitBlock(builders.BlockPair().WithHeight(blockHeight).WithBlockCreated(blockCreated).Build())

	require.NoError(t, err)
	require.EqualValues(t, driver.numOfWrittenBlocks(), 1)

	driver.verifyMocks()

	lastCommittedBlockHeight := driver.getLastBlockHeight()

	require.EqualValues(t, lastCommittedBlockHeight.LastCommittedBlockHeight, primitives.BlockHeight(blockHeight))
	require.EqualValues(t, lastCommittedBlockHeight.LastCommittedBlockTimestamp, primitives.TimestampNano(blockCreated.UnixNano()))

	// TODO Spec: If any of the intra block syncs (StateStorage, TransactionPool) is blocking and waiting, wake it up.
}

func TestCommitBlockDoesNotUpdateCommittedBlockHeightAndTimestampIfStorageFails(t *testing.T) {
	driver := NewDriver()
	driver.t = t

	driver.expectCommitStateDiff()

	blockCreated := time.Now()
	blockHeight := primitives.BlockHeight(1)

	driver.commitBlock(builders.BlockPair().WithHeight(blockHeight).WithBlockCreated(blockCreated).Build())
	require.EqualValues(t, driver.numOfWrittenBlocks(), 1)

	driver.failNextBlocks()
	driver.expectCommitStateDiff() // TODO: this line should be removed, it's added here due to convoluted sync mechanism in acceptance test where we wait until block is written to block persistence where instead we need to wait on block written to state persistence

	_, err := driver.commitBlock(builders.BlockPair().WithHeight(blockHeight + 1).Build())
	require.EqualError(t, err, "could not write a block")

	driver.verifyMocks()

	lastCommittedBlockHeight := driver.getLastBlockHeight()

	require.EqualValues(t, lastCommittedBlockHeight.LastCommittedBlockHeight, blockHeight)
	require.EqualValues(t, lastCommittedBlockHeight.LastCommittedBlockTimestamp, blockCreated.UnixNano())
}

func TestCommitBlockReturnsErrorWhenProtocolVersionMismatches(t *testing.T) {
	driver := NewDriver()
	driver.t = t

	_, err := driver.commitBlock(builders.BlockPair().WithProtocolVersion(99999).Build())

	require.EqualError(t, err, "protocol version mismatch")
}

func TestCommitBlockDiscardsBlockIfAlreadyExists(t *testing.T) {
	driver := NewDriver()
	driver.t = t

	blockPair := builders.BlockPair().Build()

	driver.expectCommitStateDiff()

	driver.commitBlock(blockPair)
	_, err := driver.commitBlock(blockPair)

	require.NoError(t, err)

	require.EqualValues(t, driver.numOfWrittenBlocks(), 1)
	driver.verifyMocks()
}

func TestCommitBlockReturnsErrorIfBlockExistsButIsDifferent(t *testing.T) {
	driver := NewDriver()
	driver.t = t

	driver.expectCommitStateDiff()

	blockPair := builders.BlockPair()

	driver.commitBlock(blockPair.Build())

	_, err := driver.commitBlock(blockPair.WithBlockCreated(time.Now().Add(1 * time.Hour)).Build())

	require.EqualError(t, err, "block already in storage, timestamp mismatch")
	require.EqualValues(t, driver.numOfWrittenBlocks(), 1)
	driver.verifyMocks()
}

func TestCommitBlockReturnsErrorIfBlockIsNotSequential(t *testing.T) {
	driver := NewDriver()
	driver.t = t
	driver.expectCommitStateDiff()

	driver.commitBlock(builders.BlockPair().Build())

	_, err := driver.commitBlock(builders.BlockPair().WithHeight(1000).Build())
	require.EqualError(t, err, "block height is 1000, expected 2")
	require.EqualValues(t, driver.numOfWrittenBlocks(), 1)
	driver.verifyMocks()
}
