package test

import (
	"testing"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
		"github.com/stretchr/testify/require"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/services/transactionpool"
	"github.com/orbs-network/orbs-network-go/instrumentation"
)

type harness struct {
	txpool services.TransactionPool
	gossip *gossiptopics.MockTransactionRelay
}

func (h *harness) expectTransactionToBeForwarded(tx *protocol.SignedTransaction) {

	h.gossip.When("BroadcastForwardedTransactions", &gossiptopics.ForwardedTransactionsInput{
		Message: &gossipmessages.ForwardedTransactionsMessage{
			SignedTransactions: []*protocol.SignedTransaction{tx},
		},
	}).Return(&gossiptopics.EmptyOutput{}, nil).Times(1)
}

func (h *harness) addNewTransaction(tx *protocol.SignedTransaction) error {
	_, err := h.txpool.AddNewTransaction(&services.AddNewTransactionInput{
		SignedTransaction: tx,
	})

	return err
}

func (h *harness) verifyMocks() error {
	_, err := h.gossip.Verify()
	return err
}

func NewHarness() *harness {
	gossip := &gossiptopics.MockTransactionRelay{}
	gossip.When("RegisterTransactionRelayHandler", mock.Any).Return()
	service := transactionpool.NewTransactionPool(gossip, instrumentation.GetLogger())

	return &harness{txpool:service, gossip:gossip}
}

func (h *harness) expectNoTransactionsToBeForwarded() {
	h.gossip.Never("BroadcastForwardedTransactions", mock.Any)
}

func TestForwardsANewValidTransactionUsingGossip(t *testing.T) {
	h := NewHarness()

	tx := builders.TransferTransaction().Build()
	h.expectTransactionToBeForwarded(tx)

	err := h.addNewTransaction(tx)

	require.NoError(t, err, "a valid transaction was not added to pool")
	require.NoError(t, h.verifyMocks(), "mock gossip was not called as expected")
}

func TestDoesNotForwardInvalidTransactionsUsingGossip(t *testing.T) {
	h := NewHarness()

	tx := builders.TransferTransaction().WithInvalidContent().Build()
	h.expectNoTransactionsToBeForwarded()

	err := h.addNewTransaction(tx)

	require.Error(t, err, "an invalid transaction was added to the pool")
	require.NoError(t, h.verifyMocks(), "mock gossip was not called (as expected)")

}
