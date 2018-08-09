package test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

var _ = Describe("Block Storage", func() {
	When("fetching transaction receipts", func() {
		Context("and the transaction was not found", func() {
			It("returns an empty receipt along with the last committed block height and timestamp", func() {
				driver := NewDriver()
				driver.expectCommitStateDiff()

				block := builders.BlockPair().Build()
				driver.commitBlock(block)

				out, err := driver.blockStorage.GetTransactionReceipt(&services.GetTransactionReceiptInput{
					Txhash:               []byte("will-not-be-found"),
					TransactionTimestamp: primitives.TimestampNano(time.Now().UnixNano()),
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(out.TransactionReceipt).To(BeNil())
				Expect(out.BlockHeight).To(BeEquivalentTo(1))
				Expect(out.BlockTimestamp).To(Equal(block.ResultsBlock.Header.Timestamp()))
			})

			XIt("while the transaction timestamp is in the future (and too far ahead to be in the grace)", func() {

			})
		})

		Context("and the transaction is found", func() {
			It("while the transaction timestamp is within the grace", func() {
				driver := NewDriver()
				driver.expectCommitStateDiff()

				block := builders.BlockPair().WithTransactions(10).WithReceiptsForTransactions().WithTimestampBloomFilter().WithTimestampNow().Build()
				driver.commitBlock(block)

				// it will be similar data transactions, but with different time stamps (and hashes..)
				block2 := builders.BlockPair().WithTransactions(10).WithReceiptsForTransactions().WithTimestampBloomFilter().WithTimestampNow().Build()
				driver.commitBlock(block2)

				// taking a transaction at 'random' (they were created at random)
				t := block.TransactionsBlock.SignedTransactions[3].Transaction()
				txHash := digest.CalcTxHash(t)

				// the block timestamp is just a couple of nanos ahead of the transactions, which is inside the grace
				out, err := driver.blockStorage.GetTransactionReceipt(&services.GetTransactionReceiptInput{
					Txhash:               txHash,
					TransactionTimestamp: t.Timestamp(),
				})

				Expect(err).ToNot(HaveOccurred())
				Expect(out.TransactionReceipt).ToNot(BeNil())
				Expect(out.TransactionReceipt.Txhash()).To(Equal(txHash))
				Expect(out.BlockHeight).To(BeEquivalentTo(1))
				Expect(out.BlockTimestamp).To(Equal(block.ResultsBlock.Header.Timestamp()))
			})

			XIt("while the transaction timestamp is outside the grace (regular)", func() {

			})

			XIt("while the transaction timestamp is at the expire window", func() {

			})

			XIt("while the transaction timestamp is at the expire window and within the grace", func() {

			})
		})
	})

})
