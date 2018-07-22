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
	panic("Not implemented")
}

func (s *service) RunLocalMethod(input *services.RunLocalMethodInput) (*services.RunLocalMethodOutput, error) {
	results, _ := s.stateStorage.ReadKeys(&services.ReadKeysInput{ContractName: "BenchmarkToken", Keys: []primitives.Ripmd160Sha256{primitives.Ripmd160Sha256("balance")}})
	sum := uint64(0)
	for _, t := range results.StateRecords {
		sum += binary.LittleEndian.Uint64(t.Value())
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
