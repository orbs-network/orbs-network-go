package contracts

import (
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

func NewVirtualMachine() services.VirtualMachine {
	return &virtualMachine {}
}

type virtualMachine struct {

}

func (vm *virtualMachine) ProcessTransactionSet(input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error) {
	panic("Not implemented")
}

func (vm *virtualMachine) RunLocalMethod(input *services.RunLocalMethodInput) (*services.RunLocalMethodOutput, error) {
	panic("Not implemented")
}

func (vm *virtualMachine) TransactionSetPreOrder(input *services.TransactionSetPreOrderInput) (*services.TransactionSetPreOrderOutput, error) {
	panic("Not implemented")
}

func (vm *virtualMachine) HandleSdkCall(input *handlers.HandleSdkCallInput) (*handlers.HandleSdkCallOutput, error) {
	panic("Not implemented")
}