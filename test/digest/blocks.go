package digest

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func MockProcessTransactionSetThatReturns(err error) func(ctx context.Context, input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error) {
	someEmptyTxSetThatWeReturnOnlyToPreventErrors := &services.ProcessTransactionSetOutput{
		TransactionReceipts: nil,
		ContractStateDiffs:  nil,
	}
	return func(ctx context.Context, input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error) {
		return someEmptyTxSetThatWeReturnOnlyToPreventErrors, err
	}
}

func MockCalcReceiptsMerkleRootThatReturns(root primitives.Sha256, err error) func(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error) {
	return func(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error) {
		return root, err
	}
}

func MockCalcStateDiffHashThatReturns(root primitives.Sha256, err error) func(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error) {
	return func(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error) {
		return root, err
	}
}
