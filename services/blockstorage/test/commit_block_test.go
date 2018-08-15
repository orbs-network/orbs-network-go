package test

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCommitBlockSavesToPersistentStorage(t *testing.T) {
	driver := NewDriver(t)

	driver.expectCommitStateDiff()

	blockCreated := time.Now()
	blockHeight := primitives.BlockHeight(1)

	_, err := driver.commitBlock(builders.BlockPair().WithHeight(blockHeight).WithBlockCreated(blockCreated).Build())

	require.NoError(t, err)
	require.EqualValues(t, 1, driver.numOfWrittenBlocks())

	driver.verifyMocks()

	lastCommittedBlockHeight := driver.getLastBlockHeight()

	require.EqualValues(t, blockHeight, lastCommittedBlockHeight.LastCommittedBlockHeight, "block height in storage should be the same")
	require.EqualValues(t, blockCreated.UnixNano(), lastCommittedBlockHeight.LastCommittedBlockTimestamp, "timestampe in storage should be the same")

	// TODO Spec: If any of the intra block syncs (StateStorage, TransactionPool) is blocking and waiting, wake it up.
}

func TestCommitBlockDoesNotUpdateCommittedBlockHeightAndTimestampIfStorageFails(t *testing.T) {
	driver := NewDriver(t)

	driver.expectCommitStateDiff()

	blockCreated := time.Now()
	blockHeight := primitives.BlockHeight(1)

	driver.commitBlock(builders.BlockPair().WithHeight(blockHeight).WithBlockCreated(blockCreated).Build())
	require.EqualValues(t, 1, driver.numOfWrittenBlocks())

	driver.failNextBlocks()
	driver.expectCommitStateDiff() // TODO: this line should be removed, it's added here due to convoluted sync mechanism in acceptance test where we wait until block is written to block persistence where instead we need to wait on block written to state persistence

	_, err := driver.commitBlock(builders.BlockPair().WithHeight(blockHeight + 1).Build())
	require.EqualError(t, err, "could not write a block", "error should be returned if storage fails")

	driver.verifyMocks()

	lastCommittedBlockHeight := driver.getLastBlockHeight()

	require.EqualValues(t, blockHeight, lastCommittedBlockHeight.LastCommittedBlockHeight, "block height should not update as storage was unavailable")
	require.EqualValues(t, blockCreated.UnixNano(), lastCommittedBlockHeight.LastCommittedBlockTimestamp, "timestamp should not update as storage was unavailable")
}

func TestCommitBlockReturnsErrorWhenProtocolVersionMismatches(t *testing.T) {
	driver := NewDriver(t)

	_, err := driver.commitBlock(builders.BlockPair().WithProtocolVersion(99999).Build())

	require.EqualError(t, err, "protocol version mismatch")
}

func TestCommitBlockDiscardsBlockIfAlreadyExists(t *testing.T) {
	driver := NewDriver(t)

	blockPair := builders.BlockPair().Build()

	driver.expectCommitStateDiff()

	driver.commitBlock(blockPair)
	_, err := driver.commitBlock(blockPair)

	require.NoError(t, err)

	require.EqualValues(t, 1, driver.numOfWrittenBlocks(), "block should be written only once")
	driver.verifyMocks()
}

func TestCommitBlockReturnsErrorIfBlockExistsButIsDifferent(t *testing.T) {
	driver := NewDriver(t)

	driver.expectCommitStateDiff()

	blockPair := builders.BlockPair()

	driver.commitBlock(blockPair.Build())

	_, err := driver.commitBlock(blockPair.WithBlockCreated(time.Now().Add(1 * time.Hour)).Build())

	require.EqualError(t, err, "block already in storage, timestamp mismatch", "same block, different timestamp should return an error")
	require.EqualValues(t, 1, driver.numOfWrittenBlocks(), "only one block should have been written")
	driver.verifyMocks()
}

func TestCommitBlockReturnsErrorIfBlockIsNotSequential(t *testing.T) {
	driver := NewDriver(t)
	driver.expectCommitStateDiff()

	driver.commitBlock(builders.BlockPair().Build())

	_, err := driver.commitBlock(builders.BlockPair().WithHeight(1000).Build())
	require.EqualError(t, err, "block height is 1000, expected 2", "block height was mutate to be invalid, should return an error")
	require.EqualValues(t, 1, driver.numOfWrittenBlocks(), "only one block should have been written")
	driver.verifyMocks()
}
