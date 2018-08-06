package test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ bool = Describe("Reading a Key", func() {
	When("key doesn't exist", func() {
		It("Returns an empty byte array", func() {
			d := newStateStorageDriver(1)
			d.commitValuePairs("fooContract", "fooKey", "fooValue")

			value, err := d.readSingleKey("fooContract", "someKey")
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal([]byte{}))
		})
	})

	When("State has only One Contract", func() {
		When("key exist", func() {
			It("Returns a Value", func() {
				value := "bar"
				key := "foo"
				contract := "some-contract"

				d := newStateStorageDriver(1)
				d.commitValuePairs(contract, key, value, "someOtherKey", value)

				output, err := d.readSingleKey(contract, key)
				Expect(err).ToNot(HaveOccurred())
				Expect(output).To(BeEquivalentTo(value))
			})
		})

		When("read 5 keys some are not existing", func() {
			It("Returns 5 values (some are empty)", func() {
				d := newStateStorageDriver(1)

				d.commitValuePairs("contract", "key1", "bar1", "key2", "bar2", "key3", "bar3", "key4", "bar4", "key5", "bar5")

				output, err := d.readKeys("contract", "key1", "key22", "key5", "key3", "key6")
				Expect(err).ToNot(HaveOccurred())
				Expect(len(output)).To(BeEquivalentTo(5))
				Expect(output[0].key).To(BeEquivalentTo("key1"))
				Expect(output[0].value).To(BeEquivalentTo("bar1"))
				Expect(output[1].key).To(BeEquivalentTo("key22"))
				Expect(output[1].value).To(Equal([]byte{}))
				Expect(output[2].key).To(BeEquivalentTo("key5"))
				Expect(output[2].value).To(BeEquivalentTo("bar5"))
				Expect(output[3].key).To(BeEquivalentTo("key3"))
				Expect(output[3].value).To(BeEquivalentTo("bar3"))
				Expect(output[4].key).To(BeEquivalentTo("key6"))
				Expect(output[4].value).To(Equal([]byte{}))
			})
		})
	})

	When("State has multiple Contracts", func() {
		When("same key exist in two contracts", func() {
			It("Returns a different Value", func() {
				key := "foo"
				v1, v2 := "bar", "bar2"

				d := newStateStorageDriver(5)

				d.commitValuePairs("contract1", key, v1)
				d.commitValuePairs("contract2", key, v2)

				output, err := d.readSingleKey("contract1", key)
				Expect(err).ToNot(HaveOccurred())
				Expect(output).To(BeEquivalentTo(v1))

				output2, err2 := d.readSingleKey("contract2", key)
				Expect(err2).ToNot(HaveOccurred())
				Expect(output2).To(BeEquivalentTo(v2))
			})
		})
	})

	When("Reading with block height states", func() {
		When("Reading inside the allowed and existing history", func() {
			It("Reads the correct past key", func() {
				key := "foo"
				v1, v2 := "bar", "bar2"

				d := newStateStorageDriver(5)
				d.commitValuePairsAtHeight(1, "contract", key, v1)
				d.commitValuePairsAtHeight(2, "contract", key, v2)

				historical, err := d.readSingleKeyFromHistory(1, "contract", key)
				Expect(err).ToNot(HaveOccurred())
				Expect(historical).To(BeEquivalentTo(v1))

				current, err := d.readSingleKeyFromHistory(2, "contract", key)
				Expect(err).ToNot(HaveOccurred())
				Expect(current).To(BeEquivalentTo(v2))
			})
		})
		When("Reading outside the allowed history", func() {
			It("fails", func() {
				key := "foo"

				d := newStateStorageDriver(1)
				d.commitValuePairsAtHeight(1, "contract", key, "bar")
				d.commitValuePairsAtHeight(2, "contract", key, "foo")

				_, err := d.readSingleKeyFromHistory(1, "contract", key)
				Expect(err).To(HaveOccurred())
			})
		})
		When("Reading in the far future", func() {
			It("fails", func() {
				key := "foo"

				d := newStateStorageDriver(1)
				d.commitValuePairsAtHeight(1, "contract", key, "bar")

				_, err := d.readSingleKeyFromHistory(100, "contract", key)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
