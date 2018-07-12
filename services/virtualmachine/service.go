package virtualmachine

import (
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

type service struct {
	blockStorage     services.BlockStorage
	blockPersistence adapter.BlockPersistence
}

func NewVirtualMachine(blockStorage services.BlockStorage, blockPersistence adapter.BlockPersistence) services.VirtualMachine {
	return &service{
		blockStorage:     blockStorage,
		blockPersistence: blockPersistence,
	}
}

func (s *service) ProcessTransactionSet(input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error) {
	panic("Not implemented")
}

func (s *service) RunLocalMethod(input *services.RunLocalMethodInput) (*services.RunLocalMethodOutput, error) {
	sum := uint64(0)
	for _, t := range s.blockPersistence.ReadAllBlocks() {
		for i := t.TransactionsBlock().SignedTransactionsOpaqueIterator(); i.HasNext(); {
			t := protocol.SignedTransactionReader(i.NextSignedTransactionsOpaque())
			sum += t.Transaction().InputArgumentsIterator().NextInputArguments().Uint64Value()
		}
	}
	arg := &protocol.MethodArgumentBuilder{Name: "balance", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: sum}
	output := &services.RunLocalMethodOutput{OutputArguments: []*protocol.MethodArgument{arg.Build()}}
	return output, nil
}

func (s *service) TransactionSetPreOrder(input *services.TransactionSetPreOrderInput) (*services.TransactionSetPreOrderOutput, error) {
	panic("Not implemented")
}

func (s *service) HandleSdkCall(input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
	panic("Not implemented")
}
