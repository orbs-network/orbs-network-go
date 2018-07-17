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

func buildContainer(height primitives.BlockHeight, timestamp primitives.Timestamp) *protocol.BlockPairContainer {
	txHeaderBuilder := protocol.TransactionsBlockHeaderBuilder{
		BlockHeight: height,
		Timestamp: timestamp,
	}

	methodArgument := protocol.MethodArgumentBuilder{Name: "date", StringValue: "1972-12-22"}

	txBuilder := protocol.TransactionBuilder{
		ContractName: "music-gig",
		MethodName: "purchase-tickets",
		InputArguments: []*protocol.MethodArgumentBuilder{
			&methodArgument,
		},
	}

	txSignedTransactionBuilder := protocol.SignedTransactionBuilder{
		Transaction: &txBuilder,
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
}

var _ = Describe("LevelDb persistence", func() {
	When("#WriteBlock", func() {
		It("does not fail", func() {
			config := NewLevelDbBlockPersistenceConfig("node1")
			db := NewLevelDbBlockPersistence(config)

			container0 := buildContainer(0, 1000)
			container1 := buildContainer(1, 2000)

			db.WriteBlock(container0)
			db.WriteBlock(container1)

			allBlocks := db.ReadAllBlocks()

			compareContainers(container0, allBlocks[0])
			compareContainers(container1, allBlocks[1])
		})
	})
})
