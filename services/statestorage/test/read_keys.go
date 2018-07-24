package test

import (
	. "github.com/onsi/ginkgo"
	"github.com/orbs-network/orbs-network-go/services/statestorage"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-network-go/test/builders"
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

var _ = Describe("Commit a State Diff", func() {

	It("persists the state into storage", func() {
		d := newStateStorageDriver()

		contract1 := builders.ContractStateDiff().WithContractName("contract1").WithStringRecord("key1", "v1").WithStringRecord("key2", "v2").Build()
		contract2 := builders.ContractStateDiff().WithContractName("contract2").WithStringRecord("key1", "v3").Build()

		d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(1).WithDiff(contract1).WithDiff(contract2).Build())

		output, err := d.readSingleKey("contract1", "key1")
		Expect(err).ToNot(HaveOccurred())
		Expect(output).To(Equal([]byte("v1")))
		output2, err := d.readSingleKey("contract1", "key2")
		Expect(err).ToNot(HaveOccurred())
		Expect(output2).To(Equal([]byte("v2")))
		output3, err := d.readSingleKey("contract2", "key1")
		Expect(err).ToNot(HaveOccurred())
		Expect(output3).To(Equal([]byte("v3")))

	})

	When("block height is not monotonously increasing", func() {
		When("too high", func() {
			It("does nothing and return desired height", func() {
				d := newStateStorageDriver()

				diff := builders.ContractStateDiff().WithContractName("contract1").WithStringRecord("key1", "whatever").Build()
				result, err := d.service.CommitStateDiff(CommitStateDiff().WithBlockHeight(3).WithDiff(diff).Build())
				Expect(err).ToNot(HaveOccurred())
				Expect(result.NextDesiredBlockHeight).To(Equal(primitives.BlockHeight(1)))

				_, err = d.readSingleKey("contract1", "key1")
				Expect(err).To(HaveOccurred())
			})
		})
		When("too low", func() {
			It("does nothing and return desired height", func() {
				d := newStateStorageDriver()
				v1 := "v1"
				v2 := "v2"

				contractDiff := builders.ContractStateDiff().WithContractName("contract1")
				diffAtHeight1 := CommitStateDiff().WithBlockHeight(1).WithDiff(contractDiff.WithStringRecord("key1", v1).Build()).Build()
				diffAtHeight2 := CommitStateDiff().WithBlockHeight(2).WithDiff(contractDiff.WithStringRecord("key1", v2).Build()).Build()

				d.service.CommitStateDiff(diffAtHeight1)
				d.service.CommitStateDiff(diffAtHeight2)

				result, err := d.service.CommitStateDiff(diffAtHeight1)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.NextDesiredBlockHeight).To(Equal(primitives.BlockHeight(3)))

				output, err := d.readSingleKey("contract1", "key1")
				Expect(err).ToNot(HaveOccurred())
				Expect(output).To(Equal([]byte(v2)))
			})
		})
	})
})


type driver struct {
	service     services.StateStorage
	persistence adapter.StatePersistence
}

func (d *driver) readSingleKey(contract string, key string) ([]byte, error) {
	out, err := d.service.ReadKeys(&services.ReadKeysInput{ContractName: primitives.ContractName(contract), Keys: []primitives.Ripmd160Sha256{[]byte(key)}})

	if err != nil {
		return nil, err
	}

	if l := len(out.StateRecords); l != 1 {
		panic(fmt.Sprintf("expected exactly one element in array. found %v", l))
	}

	if actual, expected := out.StateRecords[0].Key(), []byte(key); !bytes.Equal(actual, expected) {
		panic(fmt.Sprintf("expected output key %s to match input key %s", actual, expected))
	}

	return out.StateRecords[0].Value(), nil
}

func (d *driver) write(contract string, key string, value []byte) {
	d.persistence.WriteState(primitives.ContractName(contract), (&protocol.StateRecordBuilder{Key: []byte(key), Value: value}).Build())
}

func newStateStorageDriver() *driver {
	p := adapter.NewInMemoryStatePersistence(&struct{}{})

	return &driver{persistence: p, service: statestorage.NewStateStorage(p)}
}
