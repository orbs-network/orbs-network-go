package test

import (
	"errors"
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"testing"
)

func (h *harness) expectNewBlockProposalRequestedAndSaved(expectedBlockHeight primitives.BlockHeight) {
	txRequestMatcher := func(i interface{}) bool {
		input, ok := i.(*services.RequestNewTransactionsBlockInput)
		return ok && input.BlockHeight.Equal(expectedBlockHeight)
	}
	rxRequestMatcher := func(i interface{}) bool {
		input, ok := i.(*services.RequestNewResultsBlockInput)
		return ok && input.BlockHeight.Equal(expectedBlockHeight)
	}
	blockHeightMatcher := func(i interface{}) bool {
		input, ok := i.(*services.CommitBlockInput)
		return ok &&
			input.BlockPair.TransactionsBlock.Header.BlockHeight().Equal(expectedBlockHeight) &&
			input.BlockPair.ResultsBlock.Header.BlockHeight().Equal(expectedBlockHeight)
	}

	builtBlockForReturn := builders.BenchmarkConsensusBlockPair().WithHeight(expectedBlockHeight).Build()
	txReturn := &services.RequestNewTransactionsBlockOutput{
		TransactionsBlock: builtBlockForReturn.TransactionsBlock,
	}
	rxReturn := &services.RequestNewResultsBlockOutput{
		ResultsBlock: builtBlockForReturn.ResultsBlock,
	}

	h.consensusContext.When("RequestNewTransactionsBlock", mock.AnyIf(fmt.Sprintf("BlockHeight equals %d", expectedBlockHeight), txRequestMatcher)).Return(txReturn, nil).Times(1)
	h.consensusContext.When("RequestNewResultsBlock", mock.AnyIf(fmt.Sprintf("BlockHeight equals %d", expectedBlockHeight), rxRequestMatcher)).Return(rxReturn, nil).Times(1)
	h.blockStorage.When("CommitBlock", mock.AnyIf(fmt.Sprintf("BlockHeight equals %d", expectedBlockHeight), blockHeightMatcher)).Return(nil, nil).Times(1)
}

func (h *harness) verifyNewBlockProposalRequestedAndSaved(t *testing.T) {
	err := test.EventuallyVerify(h.consensusContext, h.blockStorage)
	if err != nil {
		t.Fatal("Did not create block with ConsensusContext or save the block to block storage:", err)
	}
}

func (h *harness) expectNewBlockProposalRequestedToFail() {
	h.consensusContext.When("RequestNewTransactionsBlock", mock.Any).Return(nil, errors.New("consensusContext error")).AtLeast(1)
	h.consensusContext.When("RequestNewResultsBlock", mock.Any).Return(nil, errors.New("consensusContext error")).Times(0)
	h.blockStorage.When("CommitBlock", mock.Any).Return(nil, nil).Times(0)
}

func (h *harness) verifyNewBlockProposalRequestedAndNotSaved(t *testing.T) {
	err := test.EventuallyVerify(h.consensusContext)
	if err != nil {
		t.Fatal("Did not create block with ConsensusContext:", err)
	}
	err = test.ConsistentlyVerify(h.blockStorage)
	if err != nil {
		t.Fatal("Did save the block to block storage:", err)
	}
}

func (h *harness) expectNewBlockProposalNotRequested() {
	h.consensusContext.When("RequestNewTransactionsBlock", mock.Any).Return(nil, errors.New("consensusContext error")).Times(0)
	h.consensusContext.When("RequestNewResultsBlock", mock.Any).Return(nil, errors.New("consensusContext error")).Times(0)
}

func (h *harness) verifyNewBlockProposalNotRequested(t *testing.T) {
	err := test.ConsistentlyVerify(h.consensusContext)
	if err != nil {
		t.Fatal("Did create block with ConsensusContext:", err)
	}
}
