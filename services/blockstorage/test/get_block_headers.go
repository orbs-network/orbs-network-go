package test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

var _ = Describe("Block storage", func() {
	When("asked to provide transactions block header", func() {
		It("returns transactions block header", func() {
			driver := NewDriver()
			driver.expectCommitStateDiff()

			block := builders.BlockPair().Build()
			driver.commitBlock(block)

			output, err := driver.blockStorage.GetTransactionsBlockHeader(&services.GetTransactionsBlockHeaderInput{BlockHeight: 1})

			Expect(err).ToNot(HaveOccurred())
			Expect(output.TransactionsBlockHeader).To(Equal(block.TransactionsBlock.Header))
			Expect(output.TransactionsBlockMetadata).To(Equal(block.TransactionsBlock.Metadata))
			Expect(output.TransactionsBlockProof).To(Equal(block.TransactionsBlock.BlockProof))
		})

		It("blocks if requested block is in near future", func(done Done) {
			driver := NewDriver()
			driver.expectCommitStateDiff()

			block := builders.BlockPair().Build()
			driver.commitBlock(block)

			result := make(chan *services.GetTransactionsBlockHeaderOutput)
			blockHeightInTheFuture := primitives.BlockHeight(5)

			go func() {
				output, _ := driver.blockStorage.GetTransactionsBlockHeader(&services.GetTransactionsBlockHeaderInput{BlockHeight: blockHeightInTheFuture})
				result <- output
			}()

			for i := 2; i <= int(blockHeightInTheFuture)+1; i++ {
				driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(i)).Build())
			}

			Expect(driver.getLastBlockHeight().LastCommittedBlockHeight).To(BeEquivalentTo(blockHeightInTheFuture + 1))

			output := <-result
			Expect(output.TransactionsBlockHeader.BlockHeight()).To(Equal(blockHeightInTheFuture))

			close(done)
		}, 100)

		It("returns error if operation times out", func(done Done) {
			driver := NewDriver()
			driver.expectCommitStateDiff()

			block := builders.BlockPair().Build()
			driver.commitBlock(block)

			timeoutError := make(chan error)
			blockHeightInTheFuture := primitives.BlockHeight(5)

			go func() {
				_, err := driver.blockStorage.GetTransactionsBlockHeader(&services.GetTransactionsBlockHeaderInput{BlockHeight: blockHeightInTheFuture})
				timeoutError <- err
			}()

			for i := primitives.BlockHeight(2); i <= 4; i++ {
				driver.commitBlock(builders.BlockPair().WithHeight(i).Build())
			}

			err := <-timeoutError
			Expect(err).To(MatchError("operation timed out"))

			close(done)
		}, 100)
	})

	When("asked to provide results block header", func() {
		It("returns results block header", func() {
			driver := NewDriver()
			driver.expectCommitStateDiff()

			block := builders.BlockPair().Build()
			driver.commitBlock(block)

			output, err := driver.blockStorage.GetResultsBlockHeader(&services.GetResultsBlockHeaderInput{BlockHeight: 1})

			Expect(err).ToNot(HaveOccurred())
			Expect(output.ResultsBlockHeader).To(Equal(block.ResultsBlock.Header))
			Expect(output.ResultsBlockProof).To(Equal(block.ResultsBlock.BlockProof))
		})

		It("blocks if requested block is in near future", func(done Done) {
			driver := NewDriver()
			driver.expectCommitStateDiff()

			block := builders.BlockPair().Build()
			driver.commitBlock(block)

			result := make(chan *services.GetResultsBlockHeaderOutput)
			blockHeightInTheFuture := primitives.BlockHeight(5)

			go func() {
				output, _ := driver.blockStorage.GetResultsBlockHeader(&services.GetResultsBlockHeaderInput{BlockHeight: blockHeightInTheFuture})
				result <- output
			}()

			for i := primitives.BlockHeight(2); i <= 6; i++ {
				driver.commitBlock(builders.BlockPair().WithHeight(i).Build())
			}

			Expect(driver.getLastBlockHeight().LastCommittedBlockHeight).To(Equal(blockHeightInTheFuture + 1))

			output := <-result

			Expect(output.ResultsBlockHeader.BlockHeight()).To(Equal(blockHeightInTheFuture))

			close(done)
		}, 100)

		It("returns error if operation times out", func(done Done) {
			driver := NewDriver()
			driver.expectCommitStateDiff()

			block := builders.BlockPair().Build()
			driver.commitBlock(block)

			timeoutError := make(chan error)
			blockHeightInTheFuture := primitives.BlockHeight(5)

			go func() {
				_, err := driver.blockStorage.GetResultsBlockHeader(&services.GetResultsBlockHeaderInput{BlockHeight: blockHeightInTheFuture})
				timeoutError <- err
			}()

			for i := primitives.BlockHeight(2); i <= 4; i++ {
				driver.commitBlock(builders.BlockPair().WithHeight(i).Build())
			}

			err := <-timeoutError
			Expect(err).To(MatchError("operation timed out"))

			close(done)
		}, 100)
	})
})
