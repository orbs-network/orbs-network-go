package test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

var _ = Describe("Block storage", func() {
	When("asked to validate the block", func() {
		It("protocol version when version when valid", func() {
			driver := NewDriver()
			block := builders.BlockPair().Build()

			_, err := driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
			Expect(err).NotTo(HaveOccurred())
		})

		It("protocol version when version is invalid", func() {
			driver := NewDriver()
			block := builders.BlockPair().Build()

			block.TransactionsBlock.Header.MutateProtocolVersion(998)

			_, err := driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
			Expect(err).To(MatchError("protocol version mismatch"))

			block.ResultsBlock.Header.MutateProtocolVersion(999)

			_, err = driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
			Expect(err).To(MatchError("protocol version mismatch"))

			block.TransactionsBlock.Header.MutateProtocolVersion(999)

			_, err = driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
			Expect(err).To(MatchError("protocol version mismatch"))

			block.TransactionsBlock.Header.MutateProtocolVersion(1)

			_, err = driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
			Expect(err).To(MatchError("protocol version mismatch"))
		})

		It("height when valid", func() {
			driver := NewDriver()
			driver.expectCommitStateDiff()

			driver.commitBlock(builders.BlockPair().Build())

			block := builders.BlockPair().WithHeight(2).Build()

			_, err := driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
			Expect(err).NotTo(HaveOccurred())
		})

		It("height when invalid", func() {
			driver := NewDriver()
			driver.expectCommitStateDiff()

			driver.commitBlock(builders.BlockPair().Build())

			block := builders.BlockPair().WithHeight(2).Build()

			block.TransactionsBlock.Header.MutateBlockHeight(998)

			_, err := driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
			Expect(err).To(MatchError("block height is 998, expected 2"))

			block.ResultsBlock.Header.MutateBlockHeight(999)

			_, err = driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
			Expect(err).To(MatchError("block height is 998, expected 2"))

			block.TransactionsBlock.Header.MutateBlockHeight(999)

			_, err = driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
			Expect(err).To(MatchError("block height is 999, expected 2"))

			block.TransactionsBlock.Header.MutateProtocolVersion(1)

			_, err = driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
			Expect(err).To(MatchError("block height is 999, expected 2"))
		})

		XIt("virtual chain")
		XIt("transactions root hash")
		XIt("metadata hash")
		XIt("receipts root hash")
		XIt("state diff hash")
		XIt("block consensus")
	})
})
