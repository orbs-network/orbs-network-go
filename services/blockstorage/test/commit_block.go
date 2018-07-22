package test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	adapter2 "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

type adapterConfig struct {
}

func (c *adapterConfig) NodeId() string {
	return "node1"
}

type driver struct {
	stateStorage   *services.MockStateStorage
	storageAdapter adapter2.BlockPersistence
	blockStorage   services.BlockStorage
}

func (d *driver) expectCommitStateDiff() {
	csdOut := &services.CommitStateDiffOutput{}

	d.stateStorage.When("CommitStateDiff", mock.Any).Return(csdOut, nil).Times(1)

}

func (d *driver) verifyMocks() {
	_, err := d.stateStorage.Verify()
	Expect(err).ToNot(HaveOccurred())
}

func (d *driver) commitBlock(blockPairContainer *protocol.BlockPairContainer) (*services.CommitBlockOutput, error) {
	return d.blockStorage.CommitBlock(&services.CommitBlockInput{
		BlockPair: blockPairContainer,
	})
}

func (d *driver) numOfWrittenBlocks() int {
	return len(d.storageAdapter.ReadAllBlocks())
}

func (d *driver) getLastBlockHeight() *services.GetLastCommittedBlockHeightOutput {
	out, err := d.blockStorage.GetLastCommittedBlockHeight(&services.GetLastCommittedBlockHeightInput{})
	Expect(err).ToNot(HaveOccurred())
	return out
}

func NewDriver() *driver {
	d := &driver{}
	d.stateStorage = &services.MockStateStorage{}
	d.storageAdapter = adapter.NewInMemoryBlockPersistence(&adapterConfig{})
	d.blockStorage = blockstorage.NewBlockStorage(d.storageAdapter, d.stateStorage)

	return d
}

var _ = Describe("Committing a block", func() {
	It("saves it to persistent storage", func() {
		driver := NewDriver()

		driver.expectCommitStateDiff()

		blockCreated := time.Now()
		blockHeight := 1

		_, err := driver.commitBlock(test.BlockPairBuilder().WithHeight(blockHeight).WithBlockCreated(blockCreated).Build())

		Expect(err).ToNot(HaveOccurred())
		Expect(driver.numOfWrittenBlocks()).To(Equal(1))

		driver.verifyMocks()

		lastCommittedBlockHeight := driver.getLastBlockHeight()

		Expect(lastCommittedBlockHeight.LastCommittedBlockHeight).To(Equal(primitives.BlockHeight(blockHeight)))
		Expect(lastCommittedBlockHeight.LastCommittedBlockTimestamp).To(Equal(primitives.TimestampNano(blockCreated.UnixNano())))

		// TODO Spec: If any of the intra block syncs (StateStorage, TransactionPool) is blocking and waiting, wake it up.
	})

	Context("block is invalid", func() {
		When("protocol version mismatches", func() {
			It("returns an error", func() {
				driver := NewDriver()

				_, err := driver.commitBlock(test.BlockPairBuilder().WithProtocolVersion(99999).Build())

				Expect(err).To(MatchError("protocol version mismatch: expected 1 got 99999"))
			})
		})

		When("block already exists", func() {
			It("should be silently discarded the block if it is the exact same block", func() {
				driver := NewDriver()

				blockPair := test.BlockPairBuilder().Build()

				driver.expectCommitStateDiff()

				driver.commitBlock(blockPair)
				_, err := driver.commitBlock(blockPair)

				Expect(err).ToNot(HaveOccurred())

				Expect(driver.numOfWrittenBlocks()).To(Equal(1))
				driver.verifyMocks()
			})

			It("should panic if it is the same height but different block", func() {
				driver := NewDriver()
				driver.expectCommitStateDiff()

				blockPair := test.BlockPairBuilder()

				driver.commitBlock(blockPair.Build())

				Expect(func() {
					driver.commitBlock(blockPair.WithBlockCreated(time.Now().Add(1 * time.Hour)).Build())
				}).To(Panic())

				Expect(driver.numOfWrittenBlocks()).To(Equal(1))
				driver.verifyMocks()
			})
		})

		When("block isn't the next of last_commited_block", func() {
			It("should panic", func() {
				driver := NewDriver()
				driver.expectCommitStateDiff()

				driver.commitBlock(test.BlockPairBuilder().Build())

				Expect(func() {
					driver.commitBlock(test.BlockPairBuilder().WithHeight(1000).Build())
				}).To(Panic())

				Expect(driver.numOfWrittenBlocks()).To(Equal(1))
				driver.verifyMocks()
			})
		})
	})
})
