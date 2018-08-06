package test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
		})

		XContext("and the transaction is found", func() {
			It("while the transaction timestamp is within the grace", func() {

			})

			It("while the transaction timestamp is outside the grace", func() {

			})

			It("while the transaction timestamp is at the expire window", func() {

			})

			It("while the transaction timestamp is at the expire window and within the grace", func() {

			})

		})
	})

})
