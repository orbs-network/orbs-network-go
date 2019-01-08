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

// Mock for CalcStateDiffHash
type mockCalcStateDiffHash struct {
	calcStateDiffHash func(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error)
}

func (m *mockCalcStateDiffHash) CalcStateDiffHash(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error) {
	return m.calcStateDiffHash(stateDiffs)
}

func NewMockCalcStateDiffHashThatReturns(root primitives.Sha256, err error) digest.CalcStateDiffHashAdapter {
	return &mockCalcStateDiffHash{
		calcStateDiffHash: func(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error) {
			return root, err
		},
	}
}
