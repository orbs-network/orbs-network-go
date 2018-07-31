package test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"os"
)

func compareTxBlocks(a *protocol.TransactionsBlockContainer, b *protocol.TransactionsBlockContainer) {
	Expect(a.Header.Equal(b.Header)).To(BeTrue())
	Expect(a.SignedTransactions[0].Equal(b.SignedTransactions[0])).To(BeTrue())
	Expect(a.BlockProof.Equal(b.BlockProof)).To(BeTrue())
	Expect(a.Metadata.Equal(b.Metadata)).To(BeTrue())

}

func compareRsBlocks(a *protocol.ResultsBlockContainer, b *protocol.ResultsBlockContainer) {
	Expect(a.Header.Equal(b.Header)).To(BeTrue())
	Expect(a.BlockProof.Equal(b.BlockProof)).To(BeTrue())
	Expect(a.TransactionReceipts[0].Equal(b.TransactionReceipts[0])).To(BeTrue())
	Expect(a.ContractStateDiffs[0].Equal(b.ContractStateDiffs[0])).To(BeTrue())
}

func compareContainers(a *protocol.BlockPairContainer, b *protocol.BlockPairContainer) {
	compareTxBlocks(a.TransactionsBlock, b.TransactionsBlock)
	compareRsBlocks(a.ResultsBlock, b.ResultsBlock)
}

func prepareStorage() (adapter.BlockPersistence, []*protocol.BlockPairContainer) {
	config := adapter.NewLevelDbBlockPersistenceConfig("node1")
	db := adapter.NewLevelDbBlockPersistence(config).WithLogger(instrumentation.GetLogger(instrumentation.String("adapter", "LevelDBPersistence")))

	block1 := builders.BlockPair().WithHeight(primitives.BlockHeight(1)).Build()
	block2 := builders.BlockPair().WithHeight(primitives.BlockHeight(2)).Build()

	db.WriteBlock(block1)
	db.WriteBlock(block2)

	return db, []*protocol.BlockPairContainer{block1, block2}
}

var _ = Describe("LevelDb persistence", func() {
	BeforeEach(func() {
		os.RemoveAll("/tmp/db")
	})

	When("#WriteBlock", func() {
		It("does not fail", func() {
			db, savedBlocks := prepareStorage()

			allBlocks := db.ReadAllBlocks()

			compareContainers(savedBlocks[0], allBlocks[0])
			compareContainers(savedBlocks[1], allBlocks[1])
		})
	})

	When("#GetTransactionsBlock", func() {
		It("Reads a certain block", func() {
			db, savedBlocks := prepareStorage()

			lastTxBlock, err := db.GetTransactionsBlock(2)

			Expect(err).ToNot(HaveOccurred())
			compareTxBlocks(savedBlocks[1].TransactionsBlock, lastTxBlock)
		})
	})

	When("#GetResultsBlock", func() {
		It("Reads a certain block", func() {
			db, savedBlocks := prepareStorage()

			lastTxBlock, err := db.GetResultsBlock(2)

			Expect(err).ToNot(HaveOccurred())
			compareRsBlocks(savedBlocks[1].ResultsBlock, lastTxBlock)
		})
	})

	When("#GetLastBlockDetails", func() {
		It("returns default values", func() {
			config := adapter.NewLevelDbBlockPersistenceConfig("node1")
			db := adapter.NewLevelDbBlockPersistence(config).WithLogger(instrumentation.GetLogger(instrumentation.String("adapter", "LevelDBPersistence")))

			height, timestamp := db.GetLastBlockDetails()

			Expect(height).To(Equal(primitives.BlockHeight(0)))
			Expect(timestamp).To(Equal(primitives.TimestampNano(0)))
		})
	})
})
