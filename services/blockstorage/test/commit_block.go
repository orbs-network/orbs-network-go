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

var _ = Describe("Committing a block", func () {

	It("saves it to persistent storage", func () {
		stateStorage := &services.MockStateStorage{}
		storageAdapter := adapter.NewInMemoryBlockPersistence(&adapterConfig{})
		service := blockstorage.NewBlockStorage(storageAdapter, stateStorage)

		commitBlockInput := &services.CommitBlockInput{
			BlockPair: buildContainer(1, 1000),
		}

		csdOut := &services.CommitStateDiffOutput{}
		stateStorage.When("CommitStateDiff", mock.Any).Return(csdOut, nil).Times(1)

		_, err := service.CommitBlock(commitBlockInput)

		Expect(err).ToNot(HaveOccurred())
		Expect(len(storageAdapter.ReadAllBlocks())).To(Equal(1))
		_, err = stateStorage.Verify()
		Expect(err).ToNot(HaveOccurred())

		lastCommitedBlockHeight, err := service.GetLastCommittedBlockHeight(&services.GetLastCommittedBlockHeightInput{})

		Expect(err).ToNot(HaveOccurred())
		Expect(lastCommitedBlockHeight.LastCommittedBlockHeight).To(Equal(primitives.BlockHeight(1)))
		Expect(lastCommitedBlockHeight.LastCommittedBlockTimestamp).To(Equal(primitives.Timestamp(1000)))


	})
})