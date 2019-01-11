package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	testValidators "github.com/orbs-network/orbs-network-go/test/crypto/validators"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func aMockValidateTransactionsBlockThatReturnsSuccess(ctx context.Context, input *services.ValidateTransactionsBlockInput) (*services.ValidateTransactionsBlockOutput, error) {
	return nil, nil
}

func aMockValidateTransactionsBlockThatReturnsError(ctx context.Context, input *services.ValidateTransactionsBlockInput) (*services.ValidateTransactionsBlockOutput, error) {
	return nil, errors.New("Some error")
}

func aMockValidateResultsBlockThatReturnsSuccess(ctx context.Context, input *services.ValidateResultsBlockInput) (*services.ValidateResultsBlockOutput, error) {
	return nil, nil
}

func aMockValidateResultsBlockThatReturnsError(ctx context.Context, input *services.ValidateResultsBlockInput) (*services.ValidateResultsBlockOutput, error) {
	return nil, errors.New("Some error")
}

// We don't care about the correctness or incorrectness of inputs because we mock the functions ValidateTransactionsBlock()
// and ValidateResultsBlock() that actually test those inputs.
// We only test the glue that holds them together. These are tests for these 2 functions in the same package where they are defined.

func TestValidateBlockProposal_FailsOnErrorInTransactionsBlock(t *testing.T) {
	block := testValidators.AStructurallyValidBlock()
	prevBlock := testValidators.AStructurallyValidBlock()
	require.False(t, validateBlockProposalInternal(context.Background(), 1, ToLeanHelixBlock(block), []byte{1, 2, 3, 4}, ToLeanHelixBlock(prevBlock), &validateBlockProposalContext{
		validateTransactionsBlock: aMockValidateTransactionsBlockThatReturnsError,
		validateResultsBlock:      aMockValidateResultsBlockThatReturnsSuccess,
		logger:                    log.GetLogger(),
	}), "should return false when ValidateTransactionsBlock() returns an error")
}

func TestValidateBlockProposal_FailsOnErrorInResultsBlock(t *testing.T) {
	block := testValidators.AStructurallyValidBlock()
	prevBlock := testValidators.AStructurallyValidBlock()
	require.False(t, validateBlockProposalInternal(context.Background(), 1, ToLeanHelixBlock(block), []byte{1, 2, 3, 4}, ToLeanHelixBlock(prevBlock), &validateBlockProposalContext{
		validateTransactionsBlock: aMockValidateTransactionsBlockThatReturnsSuccess,
		validateResultsBlock:      aMockValidateResultsBlockThatReturnsError,
		logger:                    log.GetLogger(),
	}), "should return false when ValidateTransactionsBlock() returns an error")
}
