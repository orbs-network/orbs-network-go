package adapter

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"testing"
)

func TestLevelDbPersistence(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gossip Transport Contract")
}

func buildContainer(height primitives.BlockHeight, timestamp primitives.TimestampNano, artist string, date string) *protocol.BlockPairContainer {
	arguments := []*protocol.MethodArgumentBuilder{
		{Name: "artist", Type: protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE, StringValue: artist},
		{Name: "date", Type: protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE, StringValue: date},
	}

	transactionsBlock := &protocol.TransactionsBlockContainer{
		Header: (&protocol.TransactionsBlockHeaderBuilder{
			BlockHeight: height,
			Timestamp:   timestamp,
		}).Build(),
		BlockProof: (&protocol.TransactionsBlockProofBuilder{
			Type: protocol.TRANSACTIONS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		}).Build(),
		Metadata: (&protocol.TransactionsBlockMetadataBuilder{}).Build(),
		SignedTransactions: []*protocol.SignedTransaction{
			(&protocol.SignedTransactionBuilder{
				Transaction: &protocol.TransactionBuilder{
					Signer: &protocol.SignerBuilder{
						Eddsa: &protocol.EdDSA01SignerBuilder{
							SignerPublicKey: []byte("fake-public-key"),
						},
					},
					ContractName:   "music-gig",
					MethodName:     "purchase-tickets",
					InputArguments: arguments,
				},
			}).Build(),
		},
	}

	resultsBlock := &protocol.ResultsBlockContainer{
		Header: (&protocol.ResultsBlockHeaderBuilder{
			BlockHeight:            height,
			Timestamp:              timestamp,
			NumContractStateDiffs:  1,
			NumTransactionReceipts: 1,
		}).Build(),
		BlockProof: (&protocol.ResultsBlockProofBuilder{
			Type: protocol.RESULTS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		}).Build(),
		ContractStateDiffs: []*protocol.ContractStateDiff{
			(&protocol.ContractStateDiffBuilder{
				ContractName: "music-gig",
				StateDiffs: []*protocol.StateRecordBuilder{
					{Key: []byte("some-key"), Value: []byte("some-value")},
				},
			}).Build(),
		},
		TransactionReceipts: []*protocol.TransactionReceipt{
			(&protocol.TransactionReceiptBuilder{
				Txhash:          []byte("some-tx-hash"),
				ExecutionResult: protocol.EXECUTION_RESULT_SUCCESS,
				OutputArguments: arguments,
			}).Build(),
		},
	}

	container := &protocol.BlockPairContainer{
		TransactionsBlock: transactionsBlock,
		ResultsBlock:      resultsBlock,
	}

	return container
}

func compareContainers(a *protocol.BlockPairContainer, b *protocol.BlockPairContainer) {
	Expect(a.TransactionsBlock.Header.BlockHeight()).To(Equal(b.TransactionsBlock.Header.BlockHeight()))
	Expect(a.TransactionsBlock.Header.Timestamp()).To(Equal(b.TransactionsBlock.Header.Timestamp()))

	Expect(a.TransactionsBlock.Header.Raw()).To(Equal(b.TransactionsBlock.Header.Raw()))
	Expect(a.TransactionsBlock.SignedTransactions[0].Raw()).To(Equal(b.TransactionsBlock.SignedTransactions[0].Raw()))
	Expect(a.TransactionsBlock.BlockProof.Raw()).To(Equal(b.TransactionsBlock.BlockProof.Raw()))
	Expect(a.TransactionsBlock.Metadata.Raw()).To(Equal(b.TransactionsBlock.Metadata.Raw()))

	Expect(a.ResultsBlock.Header.Raw()).To(Equal(b.ResultsBlock.Header.Raw()))
	Expect(a.ResultsBlock.BlockProof.Raw()).To(Equal(b.ResultsBlock.BlockProof.Raw()))
	Expect(a.ResultsBlock.TransactionReceipts[0].Raw()).To(Equal(b.ResultsBlock.TransactionReceipts[0].Raw()))
	Expect(a.ResultsBlock.ContractStateDiffs[0].Raw()).To(Equal(b.ResultsBlock.ContractStateDiffs[0].Raw()))
}

var _ = Describe("LevelDb persistence", func() {
	When("#WriteBlock", func() {
		It("does not fail", func() {
			config := NewLevelDbBlockPersistenceConfig("node1")
			db := NewLevelDbBlockPersistence(config)

			container0 := buildContainer(0, 1000, "David Bowie", "1972-12-22")
			container1 := buildContainer(1, 2000, "Iggy Pop", "1971-12-25")

			db.WriteBlock(container0)
			db.WriteBlock(container1)

			allBlocks := db.ReadAllBlocks()

			compareContainers(container0, allBlocks[0])
			compareContainers(container1, allBlocks[1])
		})
	})
})
