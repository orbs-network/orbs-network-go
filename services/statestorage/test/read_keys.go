package test

import (
	. "github.com/onsi/ginkgo"
	"github.com/orbs-network/orbs-network-go/services/statestorage"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
)

var _ = Describe("Reading a Key", func() {
	When("not providing a contract name", func() {
		It("Returns an error", func() {
			d := newStateStorageDriver()
			_, err := d.readKey("", "someKey")
			Expect(err).To(MatchError("missing contract name"))
		})
	})

	When("key doesn't exist", func() {
		It("Returns no Value", func() {
			d := newStateStorageDriver()
			_, err := d.readKey("fooContract", "someKey")
			Expect(err).To(MatchError("no value found for input key(s)"))
		})
	})

	When ( "State has only One Contract", func() {
		When("key exist", func() {
			It("Returns a Value", func() {
				value := []byte("bar")
				key := "foo"
				contract := "some-contract"

				d := newStateStorageDriver()
				d.write(contract, key, value)
				d.write(contract, "someOtherKey", value)

				output, err := d.readKey(contract, key)

				Expect(err).ToNot(HaveOccurred())
				Expect(output).To(HaveLen(1))
				Expect(string(output[0].Key())).To(Equal(key))
				Expect(output[0].Value()).To(Equal(value))
			})
		})
	})

	When ( "State has multiple Contracts", func() {
		When("same key exist in two contracts", func() {
			It("Returns a different Value", func() {
				key := "foo"
				v1, v2 := []byte("bar"), []byte("bar2")

				d := newStateStorageDriver()
				d.write("contract1", key, v1)
				d.write("contract2", key, v2)

				output, err := d.readKey("contract1", key)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(output[0].Key())).To(Equal(key))
				Expect(output[0].Value()).To(Equal(v1))

				output2, err2 := d.readKey("contract2", key)
				Expect(err2).ToNot(HaveOccurred())
				Expect(string(output2[0].Key())).To(Equal(key))
				Expect(output2[0].Value()).To(Equal(v2))
			})
		})
	})
})

type driver struct {
	service services.StateStorage
	persistence adapter.StatePersistence
}

func (d *driver) readKey(contract string, key string) ([]*protocol.StateRecord, error) {
	out, err := d.service.ReadKeys(&services.ReadKeysInput{ContractName: primitives.ContractName(contract), Keys: []primitives.Ripmd160Sha256{[]byte(key)}})

	if err != nil {
		return nil, err
	}

	return out.StateRecords, nil
}

func (d *driver) write(contract string, key string, value []byte) {
	d.persistence.WriteState(primitives.ContractName(contract), (&protocol.StateRecordBuilder{Key: []byte(key), Value: value}).Build())
}

func newStateStorageDriver() *driver {
	p := adapter.NewInMemoryStatePersistence(&struct{}{})

	return &driver {persistence: p, service: statestorage.NewStateStorage(p)}
}


