package digest

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

// Mock for CalcReceiptsMerkleRoot
type mockCalcReceiptsMerkleRoot struct {
	calcReceiptsMerkleRoot func(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error)
}

func (m *mockCalcReceiptsMerkleRoot) CalcReceiptsMerkleRoot(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error) {
	return m.calcReceiptsMerkleRoot(receipts)
}

func NewMockCalcReceiptsMerkleRootThatReturns(root primitives.Sha256, err error) digest.CalcReceiptsMerkleRootAdapter {
	return &mockCalcReceiptsMerkleRoot{

		calcReceiptsMerkleRoot: func(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error) {
			return root, err
		},
	}
}

// Mock for CalcStateDiffMerkleRoot
type mockCalcStateDiffMerkleRoot struct {
	calcStateDiffMerkleRoot func(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error)
}

func (m *mockCalcStateDiffMerkleRoot) CalcStateDiffMerkleRoot(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error) {
	return m.calcStateDiffMerkleRoot(stateDiffs)
}

func NewMockCalcStateDiffMerkleRootThatReturns(root primitives.Sha256, err error) digest.CalcStateDiffMerkleRootAdapter {
	return &mockCalcStateDiffMerkleRoot{
		calcStateDiffMerkleRoot: func(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error) {
			return root, err
		},
	}
}
