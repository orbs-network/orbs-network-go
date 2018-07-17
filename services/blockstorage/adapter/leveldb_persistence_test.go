package adapter

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

func TestLevelDbPersistence(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gossip Transport Contract")
}

func buildContainer(height primitives.BlockHeight, timestamp primitives.Timestamp, artist string, date string) *protocol.BlockPairContainer {
	txHeaderBuilder := &protocol.TransactionsBlockHeaderBuilder{
		BlockHeight: height,
		Timestamp: timestamp,
	}

	txSignedTransactionBuilder := &protocol.SignedTransactionBuilder{
		Transaction: &protocol.TransactionBuilder{
			Signer: &protocol.SignerBuilder{
				Eddsa: &protocol.EdDSA01SignerBuilder{
					SignerPublicKey: []byte("fake-public-key"),
				},
			},
			ContractName: "music-gig",
			MethodName:   "purchase-tickets",
			InputArguments: []*protocol.MethodArgumentBuilder{
				{Name: "artist", Type: protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE, StringValue: artist},
				{Name: "date", Type: protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE, StringValue: date},
			},
		},
	}

	txBlockProofBuilder := protocol.TransactionsBlockProofBuilder{
		Type: protocol.TRANSACTIONS_BLOCK_PROOF_TYPE_LEAN_HELIX,
	}

	transactionsBlock := &protocol.TransactionsBlockContainer{
		Header: txHeaderBuilder.Build(),
		Metadata: (&protocol.TransactionsBlockMetadataBuilder{}).Build(),
		SignedTransactions: []*protocol.SignedTransaction{
			txSignedTransactionBuilder.Build(),
		},
		BlockProof: txBlockProofBuilder.Build(),
	}

	resultsBlock := &protocol.ResultsBlockContainer{}

	container := &protocol.BlockPairContainer{
		TransactionsBlock: transactionsBlock,
		ResultsBlock: resultsBlock,
	}

	return container
}

func compareContainers(a *protocol.BlockPairContainer, b *protocol.BlockPairContainer) {
	Expect(a.TransactionsBlock.Header.BlockHeight()).To(Equal(b.TransactionsBlock.Header.BlockHeight()))
	Expect(a.TransactionsBlock.Header.Timestamp()).To(Equal(b.TransactionsBlock.Header.Timestamp()))

	Expect(a.TransactionsBlock.SignedTransactions[0].Transaction().ContractName()).To(Equal(b.TransactionsBlock.SignedTransactions[0].Transaction().ContractName()))
	Expect(a.TransactionsBlock.SignedTransactions[0].Transaction().InputArgumentsIterator().NextInputArguments().StringValue()).To(Equal(b.TransactionsBlock.SignedTransactions[0].Transaction().InputArgumentsIterator().NextInputArguments().StringValue()))
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
