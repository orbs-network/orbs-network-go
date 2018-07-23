package virtualmachine

import (
	"encoding/binary"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type service struct {
	blockStorage        services.BlockStorage
	stateStorage        services.StateStorage
	processor           services.Processor           // TODO: change to a map
	crosschainConnector services.CrosschainConnector // TODO: change to a map
}

func NewVirtualMachine(
	blockStorage services.BlockStorage,
	stateStorage services.StateStorage,
	processor services.Processor,
	crosschainConnector services.CrosschainConnector,
) services.VirtualMachine {

	return &service{
		blockStorage:        blockStorage,
		processor:           processor,
		crosschainConnector: crosschainConnector,
		stateStorage:        stateStorage,
	}
}

func (s *service) ProcessTransactionSet(input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error) {
	balance := input.SignedTransactions[0].Transaction().InputArgumentsIterator().NextInputArguments().Uint64Value()

	existingState, err := s.stateStorage.ReadKeys(&services.ReadKeysInput{ContractName: "BenchmarkToken", Keys: []primitives.Ripmd160Sha256{primitives.Ripmd160Sha256("balance")}})

	if err == nil && len(existingState.StateRecords) > 0 {
		balance += binary.LittleEndian.Uint64(existingState.StateRecords[0].Value())
	}

	byteArray := make([]byte, 8)
	binary.LittleEndian.PutUint64(byteArray, balance)

	var state []*protocol.StateRecordBuilder
	transactionStateDiff := &protocol.StateRecordBuilder{
		Key:   primitives.Ripmd160Sha256("balance"),
		Value: byteArray,
	}
	state = append(state, transactionStateDiff)

	output := &services.ProcessTransactionSetOutput{ContractStateDiffs: []*protocol.ContractStateDiff{(&protocol.ContractStateDiffBuilder{StateDiffs: state, ContractName: "BenchmarkToken"}).Build()}}
	return output, nil
}

func (s *service) RunLocalMethod(input *services.RunLocalMethodInput) (*services.RunLocalMethodOutput, error) {
	sum := uint64(0)

	results, err := s.stateStorage.ReadKeys(&services.ReadKeysInput{ContractName: "BenchmarkToken", Keys: []primitives.Ripmd160Sha256{primitives.Ripmd160Sha256("balance")}})
	if err == nil {
		for _, t := range results.StateRecords {
			sum += binary.LittleEndian.Uint64(t.Value())
		}
	}

	arg := (&protocol.MethodArgumentBuilder{
		Name:        "balance",
		Type:        protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE,
		Uint64Value: sum,
	}).Build()
	return &services.RunLocalMethodOutput{
		OutputArguments: []*protocol.MethodArgument{arg},
	}, nil
}

func (s *service) TransactionSetPreOrder(input *services.TransactionSetPreOrderInput) (*services.TransactionSetPreOrderOutput, error) {
	panic("Not implemented")
}

func (s *service) HandleSdkCall(input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
	panic("Not implemented")
}
