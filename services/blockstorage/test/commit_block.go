package test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"time"
)

var _ = Describe("Committing a block", func() {
	It("saves it to persistent storage", func() {
		driver := NewDriver()

		driver.expectCommitStateDiff()

		blockCreated := time.Now()
		blockHeight := primitives.BlockHeight(1)

		_, err := driver.commitBlock(builders.BlockPair().WithHeight(blockHeight).WithBlockCreated(blockCreated).Build())

		Expect(err).ToNot(HaveOccurred())
		Expect(driver.numOfWrittenBlocks()).To(Equal(1))

		driver.verifyMocks()

		lastCommittedBlockHeight := driver.getLastBlockHeight()

		Expect(lastCommittedBlockHeight.LastCommittedBlockHeight).To(Equal(primitives.BlockHeight(blockHeight)))
		Expect(lastCommittedBlockHeight.LastCommittedBlockTimestamp).To(Equal(primitives.TimestampNano(blockCreated.UnixNano())))

		// TODO Spec: If any of the intra block syncs (StateStorage, TransactionPool) is blocking and waiting, wake it up.
	})

	It("does not update last committed block height and timestamp when persistence storage fails", func() {
		driver := NewDriver()

		driver.expectCommitStateDiff()

		blockCreated := time.Now()
		blockHeight := primitives.BlockHeight(1)

		driver.commitBlock(builders.BlockPair().WithHeight(blockHeight).WithBlockCreated(blockCreated).Build())
		Expect(driver.numOfWrittenBlocks()).To(Equal(1))

		driver.failNextBlocks()

		_, err := driver.commitBlock(builders.BlockPair().WithHeight(blockHeight + 1).Build())
		Expect(err).To(MatchError("could not write a block"))

		driver.verifyMocks()

		lastCommittedBlockHeight := driver.getLastBlockHeight()

		Expect(lastCommittedBlockHeight.LastCommittedBlockHeight).To(BeEquivalentTo(blockHeight))
		Expect(lastCommittedBlockHeight.LastCommittedBlockTimestamp).To(BeEquivalentTo(blockCreated.UnixNano()))
	})

	Context("block is invalid", func() {
		When("protocol version mismatches", func() {
			It("returns an error", func() {
				driver := NewDriver()

				_, err := driver.commitBlock(builders.BlockPair().WithProtocolVersion(99999).Build())

				Expect(err).To(MatchError("protocol version mismatch"))
			})
		})

		When("block already exists", func() {
			It("should be silently discarded the block if it is the exact same block", func() {
				driver := NewDriver()

				blockPair := builders.BlockPair().Build()

				driver.expectCommitStateDiff()

				driver.commitBlock(blockPair)
				_, err := driver.commitBlock(blockPair)

				Expect(err).ToNot(HaveOccurred())

				Expect(driver.numOfWrittenBlocks()).To(Equal(1))
				driver.verifyMocks()
			})

			It("should return an error if it is the same height but different block", func() {
				driver := NewDriver()
				driver.expectCommitStateDiff()

				blockPair := builders.BlockPair()

				driver.commitBlock(blockPair.Build())

				_, err := driver.commitBlock(blockPair.WithBlockCreated(time.Now().Add(1 * time.Hour)).Build())

				Expect(err).To(MatchError("block already in storage, timestamp mismatch"))
				Expect(driver.numOfWrittenBlocks()).To(Equal(1))
				driver.verifyMocks()
			})
		})

		When("block isn't the next of last_commited_block", func() {
			It("should return an error", func() {
				driver := NewDriver()
				driver.expectCommitStateDiff()

				driver.commitBlock(builders.BlockPair().Build())

				_, err := driver.commitBlock(builders.BlockPair().WithHeight(1000).Build())
				Expect(err).To(MatchError("block height is 1000, expected 2"))
				Expect(driver.numOfWrittenBlocks()).To(Equal(1))
				driver.verifyMocks()
			})
		})
	})
})
