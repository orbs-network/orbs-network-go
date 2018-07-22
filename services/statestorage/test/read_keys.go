package test

import (
	. "github.com/onsi/ginkgo"
	"github.com/orbs-network/orbs-network-go/services/statestorage"
	"github.com/orbs-network/orbs-network-go/test/harness/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)



var _ = Describe("Reading a Key", func() {
	When("not providing a contract name", func() {
		It("Returns an error", func() {
			stateStorage := statestorage.NewStateStorage(adapter.NewInMemoryStatePersistence(&struct{}{}))
			_, err := stateStorage.ReadKeys(&services.ReadKeysInput{Keys: []primitives.Ripmd160Sha256{[]byte{0x40}}})
			Expect(err).To(MatchError("missing contract name"))
		})
	})

	When("key doesn't exist", func() {
		It("Returns no Value", func() {
			stateStorage := statestorage.NewStateStorage(adapter.NewInMemoryStatePersistence(&struct{}{}))
			_, err := stateStorage.ReadKeys(&services.ReadKeysInput{ContractName: "fooContract", Keys: []primitives.Ripmd160Sha256{[]byte{0x40}}})
			Expect(err).To(MatchError("no value found for input key(s)"))
		})
	})

	When ( "State has only One Contract", func() {
		When("key exist", func() {
			It("Returns a Value", func() {
				key := primitives.Ripmd160Sha256("foo")
				value := []byte("bar")
				persistence := adapter.NewInMemoryStatePersistence(&struct{}{})
				persistence.WriteState("Foo", (&protocol.StateRecordBuilder{Key: key, Value: value}).Build())
				persistence.WriteState("Foo", (&protocol.StateRecordBuilder{Key: primitives.Ripmd160Sha256("foo2"), Value: value}).Build())
				stateStorage := statestorage.NewStateStorage(persistence)

				output, err := stateStorage.ReadKeys(&services.ReadKeysInput{ContractName: "Foo", Keys: []primitives.Ripmd160Sha256{key}})

				Expect(err).ToNot(HaveOccurred())
				Expect(output.StateRecords).To(HaveLen(1))
				Expect(output.StateRecords[0].Key()).To(Equal(key))
				Expect(output.StateRecords[0].Value()).To(Equal(value))
			})
		})
	})

	When ( "State has multiple Contracts", func() {
		When("same key exist in two contracts", func() {
			It("Returns a different Value", func() {
				contract, contract2 := primitives.ContractName("Foo"), primitives.ContractName("Foo2")
				key := primitives.Ripmd160Sha256("foo")
				value, value2 := []byte("bar"), []byte("bar2")
				persistence := adapter.NewInMemoryStatePersistence(&struct{}{})
				persistence.WriteState(contract, (&protocol.StateRecordBuilder{Key: key, Value: value}).Build())
				persistence.WriteState(contract, (&protocol.StateRecordBuilder{Key: primitives.Ripmd160Sha256("foo2"), Value: []byte("bar3")}).Build())
				persistence.WriteState(contract2, (&protocol.StateRecordBuilder{Key: key, Value: value2}).Build())
				persistence.WriteState(contract2, (&protocol.StateRecordBuilder{Key: primitives.Ripmd160Sha256("foo4"), Value: []byte("bar3")}).Build())
				stateStorage := statestorage.NewStateStorage(persistence)

				output, err := stateStorage.ReadKeys(&services.ReadKeysInput{ContractName: contract, Keys: []primitives.Ripmd160Sha256{key}})
				output2, err2 := stateStorage.ReadKeys(&services.ReadKeysInput{ContractName: contract2, Keys: []primitives.Ripmd160Sha256{key}})

				Expect(err).ToNot(HaveOccurred())
				Expect(output.StateRecords).To(HaveLen(1))
				Expect(output.StateRecords[0].Key()).To(Equal(key))
				Expect(output.StateRecords[0].Value()).To(Equal(value))
				Expect(err2).ToNot(HaveOccurred())
				Expect(output2.StateRecords).To(HaveLen(1))
				Expect(output2.StateRecords[0].Key()).To(Equal(key))
				Expect(output2.StateRecords[0].Value()).To(Equal(value2))
			})
		})
	})
})
