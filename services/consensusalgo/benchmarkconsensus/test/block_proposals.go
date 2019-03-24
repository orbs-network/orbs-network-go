// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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

// expectations

func (h *harness) expectNewBlockProposalRequestedAndSaved(expectedBlockHeight primitives.BlockHeight) {
	txRequestMatcher := func(i interface{}) bool {
		input, ok := i.(*services.RequestNewTransactionsBlockInput)
		return ok && input.CurrentBlockHeight.Equal(expectedBlockHeight)
	}
	rxRequestMatcher := func(i interface{}) bool {
		input, ok := i.(*services.RequestNewResultsBlockInput)
		return ok && input.CurrentBlockHeight.Equal(expectedBlockHeight)
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

	h.consensusContext.When("RequestNewTransactionsBlock", mock.Any, mock.AnyIf(fmt.Sprintf("BlockHeight equals %d", expectedBlockHeight), txRequestMatcher)).Return(txReturn, nil).Times(1)
	h.consensusContext.When("RequestNewResultsBlock", mock.Any, mock.AnyIf(fmt.Sprintf("BlockHeight equals %d", expectedBlockHeight), rxRequestMatcher)).Return(rxReturn, nil).Times(1)
	h.blockStorage.When("CommitBlock", mock.Any, mock.AnyIf(fmt.Sprintf("BlockHeight equals %d", expectedBlockHeight), blockHeightMatcher)).Return(nil, nil).Times(1)
}

func (h *harness) verifyNewBlockProposalRequestedAndSaved(t *testing.T) {
	err := test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT, h.consensusContext, h.blockStorage)
	if err != nil {
		t.Fatal("Did not create block with ConsensusContext or save the block to block storage:", err)
	}
}

func (h *harness) expectNewBlockProposalRequestedToFail() {
	h.consensusContext.When("RequestNewTransactionsBlock", mock.Any, mock.Any).Return(nil, errors.New("consensusContext error")).AtLeast(1)
	h.consensusContext.When("RequestNewResultsBlock", mock.Any, mock.Any).Return(nil, errors.New("consensusContext error")).Times(0)
	h.blockStorage.When("CommitBlock", mock.Any, mock.Any).Return(nil, nil).Times(0)
}

func (h *harness) verifyNewBlockProposalRequestedAndNotSaved(t *testing.T) {
	err := test.EventuallyVerify(test.EVENTUALLY_ACCEPTANCE_TIMEOUT, h.consensusContext)
	if err != nil {
		t.Fatal("Did not create block with ConsensusContext:", err)
	}
	err = test.ConsistentlyVerify(test.CONSISTENTLY_ACCEPTANCE_TIMEOUT, h.blockStorage)
	if err != nil {
		t.Fatal("Did save the block to block storage:", err)
	}
}

func (h *harness) expectNewBlockProposalNotRequested() {
	h.consensusContext.When("RequestNewTransactionsBlock", mock.Any, mock.Any).Return(nil, errors.New("consensusContext error")).Times(0)
	h.consensusContext.When("RequestNewResultsBlock", mock.Any, mock.Any).Return(nil, errors.New("consensusContext error")).Times(0)
}

func (h *harness) verifyNewBlockProposalNotRequested(t *testing.T) {
	err := test.ConsistentlyVerify(test.CONSISTENTLY_ACCEPTANCE_TIMEOUT, h.consensusContext)
	if err != nil {
		t.Fatal("Did create block with ConsensusContext:", err)
	}
}
