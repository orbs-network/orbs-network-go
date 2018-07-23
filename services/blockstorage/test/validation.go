package test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

var _ = Describe("Block storage", func () {
	When("asked to validate transactions block", func() {
		It("checks protocol version", func() {
			driver := NewDriver()
			block := test.BlockPairBuilder().Build()

			_, err := driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
			Expect(err).NotTo(HaveOccurred())

			block.TransactionsBlock.Header.MutateProtocolVersion(999)

			_, err =  driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
			Expect(err).To(MatchError("protocol version mismatch: expected 1 got 999"))
		})

		XIt("checks virtual chain")

		It("checks block height", func () {
			driver := NewDriver()
			driver.expectCommitStateDiff()

			driver.commitBlock(test.BlockPairBuilder().Build())

			block := test.BlockPairBuilder().WithHeight(2).Build()

			_, err := driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
			Expect(err).NotTo(HaveOccurred())

			block.ResultsBlock.Header.MutateBlockHeight(999)

			_, err =  driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
			Expect(err).To(MatchError("block height is 999, expected 2"))
		})

		XIt("checks transactions root hash")

		XIt("checks metadata hash")


	})

	When("asked to validate results block", func () {
		It("checks protocol version", func() {
			driver := NewDriver()
			block := test.BlockPairBuilder().Build()

			_, err := driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
			Expect(err).NotTo(HaveOccurred())

			block.ResultsBlock.Header.MutateProtocolVersion(1000)

			_, err =  driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
			Expect(err).To(MatchError("protocol version mismatch: expected 1 got 1000"))
		})

		XIt("checks virtual chain", func () {

		})

		It("checks block height", func () {
			driver := NewDriver()
			driver.expectCommitStateDiff()

			driver.commitBlock(test.BlockPairBuilder().Build())

			block := test.BlockPairBuilder().WithHeight(2).Build()

			_, err := driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
			Expect(err).NotTo(HaveOccurred())

			block.ResultsBlock.Header.MutateBlockHeight(1000)

			_, err =  driver.blockStorage.ValidateBlockForCommit(&services.ValidateBlockForCommitInput{block})
			Expect(err).To(MatchError("block height is 1000, expected 2"))
		})


		XIt("checks receipts root hash")

		XIt("checks state diff hash")

		XIt("checks block consensus")
	})
})