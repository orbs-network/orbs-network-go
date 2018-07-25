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

	When("providing a non existing contract", func() {
		It("Returns an error", func() {
			d := newStateStorageDriver()
			_, err := d.readSingleKey("foo", "someKey")
			Expect(err).To(MatchError("missing contract name"))
		})
	})

	When("key doesn't exist", func() {
		It("Returns an empty byte array", func() {
			d := newStateStorageDriver()
			d.write("fooContract", "someRandomKeyToForceNewContract", []byte("randomValue"))
			value, err := d.readSingleKey("fooContract", "someKey")
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal([]byte{}))
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

		When("read 5 keys some are not existing", func() {
			It("Returns 5 values (some are empty)", func() {
				d := newStateStorageDriver()

				d.write("contract", "key1", []byte("bar1"))
				d.write("contract", "key2", []byte("bar2"))
				d.write("contract", "key3", []byte("bar3"))
				d.write("contract", "key4", []byte("bar4"))
				d.write("contract", "key5", []byte("bar5"))

				output, err := d.readKeys("contract", "key1", "key22", "key5", "key3", "key6")
				Expect(err).ToNot(HaveOccurred())
				Expect(len(output)).To(Equal(5))
				Expect(output[0].key).To(Equal("key1"))
				Expect(output[0].value).To(Equal([]byte("bar1")))
				Expect(output[1].key).To(Equal("key22"))
				Expect(output[1].value).To(Equal([]byte{}))
				Expect(output[2].key).To(Equal("key5"))
				Expect(output[2].value).To(Equal([]byte("bar5")))
				Expect(output[3].key).To(Equal("key3"))
				Expect(output[3].value).To(Equal([]byte("bar3")))
				Expect(output[4].key).To(Equal("key6"))
				Expect(output[4].value).To(Equal([]byte{}))
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
