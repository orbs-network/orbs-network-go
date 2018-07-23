package test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

var _ = Describe("Block storage", func () {
	When("asked to provide transactions block header", func () {
		It("returns transactions block header", func () {
			driver := NewDriver()
			driver.expectCommitStateDiff()

			block := test.BlockPairBuilder().Build()
			driver.commitBlock(block)

			output, err := driver.blockStorage.GetTransactionsBlockHeader(&services.GetTransactionsBlockHeaderInput{BlockHeight:1})

			Expect(err).ToNot(HaveOccurred())
			Expect(output.TransactionsBlockHeader).To(Equal(block.TransactionsBlock.Header))
			Expect(output.TransactionsBlockMetadata).To(Equal(block.TransactionsBlock.Metadata))
			Expect(output.TransactionsBlockProof).To(Equal(block.TransactionsBlock.BlockProof))
		})

		It("blocks if requested block is in near future", func (done Done) {
			driver := NewDriver()
			driver.expectCommitStateDiff()

			block := test.BlockPairBuilder().Build()
			driver.commitBlock(block)

			result := make(chan *services.GetTransactionsBlockHeaderOutput)

			go func () {
				output, _ := driver.blockStorage.GetTransactionsBlockHeader(&services.GetTransactionsBlockHeaderInput{BlockHeight:5})
				result <- output
			}()

			for i:=2; i <=6 ; i++ {
				driver.commitBlock(test.BlockPairBuilder().WithHeight(i).Build())
			}

			Expect(driver.getLastBlockHeight().LastCommittedBlockHeight).To(Equal(primitives.BlockHeight(6)))

			block5 := driver.getBlock(5).TransactionsBlock
			output := <- result

			Expect(output.TransactionsBlockHeader.BlockHeight()).To(Equal(primitives.BlockHeight(5)))
			Expect(output.TransactionsBlockHeader).To(Equal(block5.Header))
			Expect(output.TransactionsBlockProof).To(Equal(block5.BlockProof))
			Expect(output.TransactionsBlockMetadata).To(Equal(block5.Metadata))

			close(done)
		}, 100)
	})

	When("asked to provide results block header", func () {
		It("returns results block header", func () {
			driver := NewDriver()
			driver.expectCommitStateDiff()

			block := test.BlockPairBuilder().Build()
			driver.commitBlock(block)

			output, err := driver.blockStorage.GetResultsBlockHeader(&services.GetResultsBlockHeaderInput{BlockHeight:1})

			Expect(err).ToNot(HaveOccurred())
			Expect(output.ResultsBlockHeader).To(Equal(block.ResultsBlock.Header))
			Expect(output.ResultsBlockProof).To(Equal(block.ResultsBlock.BlockProof))
		})

		It("blocks if requested block is in near future", func (done Done) {
			driver := NewDriver()
			driver.expectCommitStateDiff()

			block := test.BlockPairBuilder().Build()
			driver.commitBlock(block)

			result := make(chan *services.GetResultsBlockHeaderOutput)

			go func () {
				output, _ := driver.blockStorage.GetResultsBlockHeader(&services.GetResultsBlockHeaderInput{BlockHeight:5})
				result <- output
			}()

			for i:=2; i <=6 ; i++ {
				driver.commitBlock(test.BlockPairBuilder().WithHeight(i).Build())
			}

			Expect(driver.getLastBlockHeight().LastCommittedBlockHeight).To(Equal(primitives.BlockHeight(6)))

			block5 := driver.getBlock(5).ResultsBlock
			output := <- result

			Expect(output.ResultsBlockHeader.BlockHeight()).To(Equal(primitives.BlockHeight(5)))
			Expect(output.ResultsBlockHeader).To(Equal(block5.Header))
			Expect(output.ResultsBlockProof).To(Equal(block5.BlockProof))

			close(done)
		}, 100)
	})
})
