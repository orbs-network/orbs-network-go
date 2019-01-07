package leanhelixconsensus

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

//// Mock for ValidateBlockFailsOnNil
//type mockValidateBlockFailsOnNilAdapter struct {
//	validateBlockFailsOnNil func(ctx context.Context, vc *validatorContext) error
//}
//
//func NewMockValidateBlockFailsOnNilThatReturns(err error) ValidateBlockFailsOnNilAdapter {
//	return &mockValidateBlockFailsOnNilAdapter{
//		validateBlockFailsOnNil: func(ctx context.Context, vc *validatorContext) error {
//			return err
//		},
//	}
//}

func TestValidateBlockFailsOnNil(t *testing.T) {
	require.Error(t, validateBlockNotNil(nil, &validatorContext{}), "fail when BlockPair is nil")

	block := &protocol.BlockPairContainer{
		TransactionsBlock: nil,
		ResultsBlock:      &protocol.ResultsBlockContainer{},
	}
	require.Error(t, validateBlockNotNil(block, &validatorContext{}), "fail when transactions block is nil")
	block.TransactionsBlock = &protocol.TransactionsBlockContainer{}
	require.Nil(t, validateBlockNotNil(block, &validatorContext{}), "ok when blockPair's transaction and results blocks are not nil")
	block.ResultsBlock = nil
	require.Error(t, validateBlockNotNil(block, &validatorContext{}), "fail when results block is nil")
}

func TestValidateBlockHash(t *testing.T) {
	t.Skip("Cannot set TransactionBlock to empty - crashes the test - fix.")
	block := &protocol.BlockPairContainer{
		TransactionsBlock: &protocol.TransactionsBlockContainer{},
		ResultsBlock:      &protocol.ResultsBlockContainer{},
	}

	require.Error(t, validateBlockHash(block, &validatorContext{}))
}
