package test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	"github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/go-mock"
	adapter2 "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
)

type adapterConfig struct {

}

func (c *adapterConfig) NodeId() string {
	return "node1"
}

func buildContainer(height primitives.BlockHeight, blockCreated primitives.Timestamp) *protocol.BlockPairContainer {
	transactionsBlock := &protocol.TransactionsBlockContainer{
		Header: (&protocol.TransactionsBlockHeaderBuilder{
			BlockHeight: height,
			Timestamp:   blockCreated,
			ProtocolVersion: blockstorage.ProtocolVersion,
		}).Build(),
		BlockProof: (&protocol.TransactionsBlockProofBuilder{
			Type: protocol.TRANSACTIONS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		}).Build(),
		Metadata: (&protocol.TransactionsBlockMetadataBuilder{}).Build(),
		SignedTransactions: []*protocol.SignedTransaction{
			(test.TransferTransaction().WithAmount(10)).Build(),
		},
	}

	resultsBlock := &protocol.ResultsBlockContainer{
		Header: (&protocol.ResultsBlockHeaderBuilder{
			BlockHeight:            height,
			Timestamp:              blockCreated,
			NumContractStateDiffs:  1,
			NumTransactionReceipts: 1,
		}).Build(),
		BlockProof: (&protocol.ResultsBlockProofBuilder{
			Type: protocol.RESULTS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		}).Build(),
		ContractStateDiffs: []*protocol.ContractStateDiff{
			(&protocol.ContractStateDiffBuilder{
				ContractName: "BenchmarkToken",
				StateDiffs: []*protocol.StateRecordBuilder{
					{ Key: []byte("amount"), Value: []byte{10} },
				},
			}).Build(),
		},
		TransactionReceipts: []*protocol.TransactionReceipt{
			(&protocol.TransactionReceiptBuilder{
				Txhash: []byte("some-tx-hash"),
				ExecutionResult: protocol.EXECUTION_RESULT_SUCCESS,
			}).Build(),
		},
	}

	container := &protocol.BlockPairContainer{
		TransactionsBlock: transactionsBlock,
		ResultsBlock: resultsBlock,
	}

	return container
}

type driver struct {
	stateStorage *services.MockStateStorage
	storageAdapter adapter2.BlockPersistence
	blockStorage services.BlockStorage
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

var _ = Describe("Committing a block", func () {
	It("saves it to persistent storage", func () {
		driver := NewDriver()

		driver.expectCommitStateDiff()

		_, err := driver.commitBlock(buildContainer(1, 1000))

		Expect(err).ToNot(HaveOccurred())
		Expect(driver.numOfWrittenBlocks()).To(Equal(1))

		driver.verifyMocks()

		lastCommittedBlockHeight := driver.getLastBlockHeight()

		Expect(lastCommittedBlockHeight.LastCommittedBlockHeight).To(Equal(primitives.BlockHeight(1)))
		Expect(lastCommittedBlockHeight.LastCommittedBlockTimestamp).To(Equal(primitives.Timestamp(1000)))

		// TODO Spec: If any of the intra block syncs (StateStorage, TransactionPool) is blocking and waiting, wake it up.
	})

	Context("block is invalid", func () {
		When("protocol version mismatches", func () {
			It("returns an error", func () {
				driver := NewDriver()

				blockPair := buildContainer(1, 1000)
				blockPair.TransactionsBlock.Header.MutateProtocolVersion(99999)

				_, err := driver.commitBlock(blockPair)

				Expect(err).To(MatchError("protocol version mismatch: expected 1 got 99999"))
			})
		})

		When("block already exists", func() {
			It("should be silently discarded the block if it is the exact same block", func () {
				driver := NewDriver()

				blockPair := buildContainer(1, 1000)

				driver.expectCommitStateDiff()

				driver.commitBlock(blockPair)
				_, err := driver.commitBlock(blockPair)

				Expect(err).ToNot(HaveOccurred())

				Expect(driver.numOfWrittenBlocks()).To(Equal(1))
				driver.verifyMocks()
			})

			It("should panic if it is the same height but different block", func () {
				driver := NewDriver()
				driver.expectCommitStateDiff()

				blockPair := buildContainer(1, 1000)
				blockPairDifferentTimestamp := buildContainer(1, 9000)

				driver.commitBlock(blockPair)

				Expect(func () {
					driver.commitBlock(blockPairDifferentTimestamp)
				}).To(Panic())

				Expect(driver.numOfWrittenBlocks()).To(Equal(1))
				driver.verifyMocks()
			})
		})
	})
})