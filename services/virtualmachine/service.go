package virtualmachine

import (
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

type service struct {
	blockStorage        services.BlockStorage
	processor           services.Processor           // TODO: change to a map
	crosschainConnector services.CrosschainConnector // TODO: change to a map
	blockPersistence    adapter.BlockPersistence
}

func NewVirtualMachine(
	blockStorage services.BlockStorage,
	processor services.Processor,
	crosschainConnector services.CrosschainConnector,
	blockPersistence adapter.BlockPersistence,
) services.VirtualMachine {

	return &service{
		blockStorage:        blockStorage,
		processor:           processor,
		crosschainConnector: crosschainConnector,
		blockPersistence:    blockPersistence,
	}
}

func (s *service) ProcessTransactionSet(input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error) {
	panic("Not implemented")
}

func (s *service) RunLocalMethod(input *services.RunLocalMethodInput) (*services.RunLocalMethodOutput, error) {
	sum := uint64(0)
	for _, blockPair := range s.blockPersistence.ReadAllBlocks() {
		for _, t := range blockPair.TransactionsBlock.SignedTransactions {
			sum += t.Transaction().InputArgumentsIterator().NextInputArguments().Uint64Value()
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
