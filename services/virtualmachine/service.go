package virtualmachine

import (
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"encoding/binary"
)

type service struct {
	blockStorage services.BlockStorage
	stateStorage services.StateStorage
}

func NewVirtualMachine(blockStorage services.BlockStorage,stateStorage services.StateStorage) services.VirtualMachine {
	return &service{
		blockStorage: blockStorage,
		stateStorage: stateStorage,
	}
}

func (s *service) ProcessTransactionSet(input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error) {
	panic("Not implemented")
}

func (s *service) RunLocalMethod(input *services.RunLocalMethodInput) (*services.RunLocalMethodOutput, error) {

	results, _ := s.stateStorage.ReadKeys(nil)
	sum := uint64(0)
	for _, t := range results.StateDiffs {
		sum += binary.LittleEndian.Uint64(t.Value())
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