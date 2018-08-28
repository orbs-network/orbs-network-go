package test

import (
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"testing"
)

func (h *harness) expectLastPersistentBlockToBeQueriedInStorage(returnLastBlockHeight primitives.BlockHeight) {
	h.blockStorage.When("GetLastCommittedBlockHeight", mock.Any).Return(&services.GetLastCommittedBlockHeightOutput{
		LastCommittedBlockHeight: returnLastBlockHeight,
	}, nil).Times(1)

	if returnLastBlockHeight > 0 {

		txHeightMatcher := func(i interface{}) bool {
			input, ok := i.(*services.GetTransactionsBlockHeaderInput)
			return ok && input.BlockHeight.Equal(returnLastBlockHeight)
		}
		rxHeightMatcher := func(i interface{}) bool {
			input, ok := i.(*services.GetResultsBlockHeaderInput)
			return ok && input.BlockHeight.Equal(returnLastBlockHeight)
		}

		builtBlockForReturn := builders.BenchmarkConsensusBlockPair().WithHeight(returnLastBlockHeight).Build()
		txReturn := &services.GetTransactionsBlockHeaderOutput{
			TransactionsBlockHeader: builtBlockForReturn.TransactionsBlock.Header,
		}
		rxReturn := &services.GetResultsBlockHeaderOutput{
			ResultsBlockHeader: builtBlockForReturn.ResultsBlock.Header,
		}

		h.blockStorage.When("GetTransactionsBlockHeader", mock.AnyIf(fmt.Sprintf("BlockHeight equals %d", returnLastBlockHeight), txHeightMatcher)).Return(txReturn, nil).Times(1)
		h.blockStorage.When("GetResultsBlockHeader", mock.AnyIf(fmt.Sprintf("BlockHeight equals %d", returnLastBlockHeight), rxHeightMatcher)).Return(rxReturn, nil).Times(1)

	}
}

func (h *harness) verifyLastPersistentBlockToBeQueriedInStorage(t *testing.T) {
	err := test.EventuallyVerify(h.blockStorage)
	if err != nil {
		t.Fatal("Did not query last persistent block from storage:", err)
	}
}
