package test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//TODO a case where we pass a set of keys to ReadKeys(), and at least one key has no matching value
var _ = Describe("Reading a Key", func() {
	When("not providing a contract name", func() {
		It("Returns an error", func() {
			d := newStateStorageDriver()
			_, err := d.readSingleKey("", "someKey")
			Expect(err).To(MatchError("missing contract name"))
		})
	})

	When("key doesn't exist", func() {
		It("Returns no Value", func() {
			d := newStateStorageDriver()
			_, err := d.readSingleKey("fooContract", "someKey")
			Expect(err).To(MatchError("no value found for input key(s)"))
		})
	})

	When("State has only One Contract", func() {
		When("key exist", func() {
			It("Returns a Value", func() {
				value := []byte("bar")
				key := "foo"
				contract := "some-contract"

				d := newStateStorageDriver()
				d.write(contract, key, value)
				d.write(contract, "someOtherKey", value)

				output, err := d.readSingleKey(contract, key)

				Expect(err).ToNot(HaveOccurred())
				Expect(output).To(Equal(value))
			})
		})
	})

	When("State has multiple Contracts", func() {
		When("same key exist in two contracts", func() {
			It("Returns a different Value", func() {
				key := "foo"
				v1, v2 := []byte("bar"), []byte("bar2")

				d := newStateStorageDriver()
				d.write("contract1", key, v1)
				d.write("contract2", key, v2)

				output, err := d.readSingleKey("contract1", key)
				Expect(err).ToNot(HaveOccurred())
				Expect(output).To(Equal(v1))

				output2, err2 := d.readSingleKey("contract2", key)
				Expect(err2).ToNot(HaveOccurred())
				Expect(output2).To(Equal(v2))
			})
		})
	})

})
