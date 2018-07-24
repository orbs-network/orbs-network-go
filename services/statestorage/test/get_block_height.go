package test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/test/builders"
)

var _ = Describe("Getting Block Height", func() {
	When("Init done", func() {
		It("Returns an block height 0", func() {
			d := newStateStorageDriver()
			height, timestamp, err := d.getBlockHeightAndTimestamp()
			Expect(err).ToNot(HaveOccurred())
			Expect(height).To(Equal(0))
			Expect(timestamp).To(Equal(0))
		})
	})

	When("Block commits", func() {
		It("Returns the current block height and timestamp", func() {
			d := newStateStorageDriver()
			heightBefore, _, err := d.getBlockHeightAndTimestamp()
			contract1 := builders.ContractStateDiff().WithContractName("contract1").WithStringRecord("key1", "v1").WithStringRecord("key2", "v2").Build()
			d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(1).WithBlockTimestamp(6579).WithDiff(contract1).Build())
			heightAfter, timestampAfter, err := d.getBlockHeightAndTimestamp()

			Expect(err).ToNot(HaveOccurred())
			Expect(heightAfter).To(Equal(heightBefore + 1))
			Expect(timestampAfter).To(Equal(6579))

		})
	})

	When("Block fails to commits", func() {
		It("Returns an block height of before", func() {
			d := newStateStorageDriver()
			contract1 := builders.ContractStateDiff().WithContractName("contract1").WithStringRecord("key1", "v1").WithStringRecord("key2", "v2").Build()
			d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(1).WithDiff(contract1).Build())
			d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(2).WithDiff(contract1).Build())
			d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(3).WithDiff(contract1).Build())
			heightBefore, _, err := d.getBlockHeightAndTimestamp()
			d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(1).WithDiff(contract1).Build())
			heightAfter, _, err := d.getBlockHeightAndTimestamp()

			Expect(err).ToNot(HaveOccurred())
			Expect(heightAfter).To(Equal(heightBefore))

		})
	})

})
