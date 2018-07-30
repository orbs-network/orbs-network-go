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
			os.RemoveAll("/tmp/db")

			config := adapter.NewLevelDbBlockPersistenceConfig("node1")
			db := adapter.NewLevelDbBlockPersistence(config).WithLogger(instrumentation.GetLogger(instrumentation.String("adapter", "LevelDBPersistence")))

			block1 := builders.BlockPair().WithHeight(primitives.BlockHeight(1)).Build()
			block2 := builders.BlockPair().WithHeight(primitives.BlockHeight(2)).Build()

			db.WriteBlock(block1)
			db.WriteBlock(block2)

			allBlocks := db.ReadAllBlocks()

			compareContainers(block1, allBlocks[0])
			compareContainers(block2, allBlocks[1])

			//FIXME does not work because of membuffers
			//Expect(allBlocks[0]).To(Equal(block1))
			//Expect(allBlocks[1]).To(Equal(block2))
		})
	})
})
