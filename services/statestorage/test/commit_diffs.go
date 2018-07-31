package test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/test/builders"
	)

var _ = Describe("Commit a State Diff", func() {

	It("persists the state into storage", func() {
		d := newStateStorageDriver(1)

		contract1 := builders.ContractStateDiff().WithContractName("contract1").WithStringRecord("key1", "v1").WithStringRecord("key2", "v2").Build()
		contract2 := builders.ContractStateDiff().WithContractName("contract2").WithStringRecord("key1", "v3").Build()

		d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(1).WithDiff(contract1).WithDiff(contract2).Build())

		output, err := d.readSingleKey("contract1", "key1")
		Expect(err).ToNot(HaveOccurred())
		Expect(output).To(BeEquivalentTo("v1"))
		output2, err := d.readSingleKey("contract1", "key2")
		Expect(err).ToNot(HaveOccurred())
		Expect(output2).To(BeEquivalentTo("v2"))
		output3, err := d.readSingleKey("contract2", "key1")
		Expect(err).ToNot(HaveOccurred())
		Expect(output3).To(BeEquivalentTo("v3"))

	})

	When("block height is not monotonously increasing", func() {
		When("too high", func() {
			It("does nothing and return desired height", func() {
				d := newStateStorageDriver(1)

				diff := builders.ContractStateDiff().WithContractName("contract1").WithStringRecord("key1", "whatever").Build()
				result, err := d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(3).WithDiff(diff).Build())
				Expect(err).ToNot(HaveOccurred())
				Expect(result.NextDesiredBlockHeight).To(BeEquivalentTo(1))

				_, err = d.readSingleKey("contract1", "key1")
				Expect(err).To(HaveOccurred())
			})
		})
		When("too low", func() {
			It("does nothing and return desired height", func() {
				d := newStateStorageDriver(1)
				v1 := "v1"
				v2 := "v2"

				contractDiff := builders.ContractStateDiff().WithContractName("contract1")
				diffAtHeight1 := CommitStateDiff().WithBlockHeight(1).WithDiff(contractDiff.WithStringRecord("key1", v1).Build()).Build()
				diffAtHeight2 := CommitStateDiff().WithBlockHeight(2).WithDiff(contractDiff.WithStringRecord("key1", v2).Build()).Build()

				d.service.CommitStateDiff(diffAtHeight1)
				d.service.CommitStateDiff(diffAtHeight2)

				diffWrongOldHeight := CommitStateDiff().WithBlockHeight(1).WithDiff(contractDiff.WithStringRecord("key1", "v3").WithStringRecord("key3", "v3").Build()).Build()
				result, err := d.service.CommitStateDiff(diffWrongOldHeight)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.NextDesiredBlockHeight).To(BeEquivalentTo(3))

				output, err := d.readSingleKeyFromHistory(2, "contract1", "key1")
				Expect(err).ToNot(HaveOccurred())
				Expect(output).To(BeEquivalentTo(v2))
				output2, err := d.readSingleKeyFromHistory(2,"contract1", "key3")
				Expect(err).ToNot(HaveOccurred())
				Expect(output2).To(BeEquivalentTo([]byte{}))
			})
		})
	})
})
