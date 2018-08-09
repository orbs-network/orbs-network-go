package test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/test/builders"
)

var _ = Describe("Getting Block Height", func() {
	When("Init done", func() {
		It("Returns an block height 0", func() {
			d := newStateStorageDriver(1)
			height, timestamp, err := d.getBlockHeightAndTimestamp()
			Expect(err).ToNot(HaveOccurred())
			Expect(height).To(Equal(0))
			Expect(timestamp).To(Equal(0))
		})
	})

	When("Block commits", func() {
		It("Returns the current block height and timestamp", func() {
			d := newStateStorageDriver(1)
			heightBefore, _, err := d.getBlockHeightAndTimestamp()
			d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(1).WithBlockTimestamp(6579).WithDiff(builders.ContractStateDiff().Build()).Build())
			heightAfter, timestampAfter, err := d.getBlockHeightAndTimestamp()

			Expect(err).ToNot(HaveOccurred())
			Expect(heightAfter).To(Equal(heightBefore + 1))
			Expect(timestampAfter).To(Equal(6579))

		})
	})

	When("Block fails to commits", func() {
		It("Returns an block height of before", func() {
			d := newStateStorageDriver(1)
			stateDiff := builders.ContractStateDiff().Build()
			d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(1).WithDiff(stateDiff).Build())
			d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(2).WithDiff(stateDiff).Build())
			heightBefore, _, err := d.getBlockHeightAndTimestamp()
			d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(1).WithDiff(stateDiff).Build())
			heightAfter, _, err := d.getBlockHeightAndTimestamp()

			Expect(err).ToNot(HaveOccurred())
			Expect(heightAfter).To(Equal(heightBefore))

		})
	})

})
