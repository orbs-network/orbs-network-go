package test

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/transactionpool"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/stretchr/testify/require"
	"testing"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
)

type harness struct {
	txpool services.TransactionPool
	gossip *gossiptopics.MockTransactionRelay
	vm     *services.MockVirtualMachine
}

func (h *harness) expectTransactionToBeForwarded(tx *protocol.SignedTransaction) {

	h.gossip.When("BroadcastForwardedTransactions", &gossiptopics.ForwardedTransactionsInput{
		Message: &gossipmessages.ForwardedTransactionsMessage{
			SignedTransactions: []*protocol.SignedTransaction{tx},
		},
	}).Return(&gossiptopics.EmptyOutput{}, nil).Times(1)
}

func (h *harness) expectNoTransactionsToBeForwarded() {
	h.gossip.Never("BroadcastForwardedTransactions", mock.Any)
}

func (h *harness) ignoringForwardMessages() {
	h.gossip.When("BroadcastForwardedTransactions", mock.Any).Return(&gossiptopics.EmptyOutput{}, nil).AtLeast(0)
}

func (h *harness) addNewTransaction(tx *protocol.SignedTransaction) error {
	_, err := h.txpool.AddNewTransaction(&services.AddNewTransactionInput{
		SignedTransaction: tx,
	})

	return err
}

func (h *harness) reportTransactionAsCommitted(transaction *protocol.SignedTransaction) {
	h.txpool.CommitTransactionReceipts(&services.CommitTransactionReceiptsInput{
		TransactionReceipts: []*protocol.TransactionReceipt{
			(&protocol.TransactionReceiptBuilder{
				Txhash: hash.CalcSha256(transaction.Raw()),
			}).Build(),
		},
	})
}

func (h *harness) verifyMocks() error {
	_, err := h.gossip.Verify()
	return err
}

func (h *harness) failPreOrderCheckFor(transaction *protocol.SignedTransaction, status protocol.TransactionStatus) {
	h.vm.When("TransactionSetPreOrder", mock.AnyIf("input with expected transaction",
		func(i interface{}) bool {
			if input, ok := i.(*services.TransactionSetPreOrderInput); !ok {
				panic("mock virtual machine invoked with bad input")
			} else if len(input.SignedTransactions) != 1 { // TODO if we need to support more than one transaction, generalize and refactor
				return false
			} else {
				return input.SignedTransactions[0].Equal(transaction)
			}

		})).Return(&services.TransactionSetPreOrderOutput{PreOrderResults: []protocol.TransactionStatus{status}}).Times(1)
}

func NewHarness() *harness {
	gossip := &gossiptopics.MockTransactionRelay{}
	gossip.When("RegisterTransactionRelayHandler", mock.Any).Return()

	virtualMachine := &services.MockVirtualMachine{}
	virtualMachine.When("TransactionSetPreOrder", mock.Any).Return(&services.TransactionSetPreOrderOutput{PreOrderResults: []protocol.TransactionStatus{protocol.TRANSACTION_STATUS_PENDING}})

	service := transactionpool.NewTransactionPool(gossip, virtualMachine, instrumentation.GetLogger())

	return &harness{txpool: service, gossip: gossip, vm: virtualMachine}
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

func TestDoesNotAddTransactionsThatFailedPreOrderChecks(t *testing.T) {
	h := NewHarness()
	tx := builders.TransferTransaction().Build()
	expectedStatus := protocol.TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER

	h.failPreOrderCheckFor(tx, expectedStatus)
	h.ignoringForwardMessages()

	err := h.addNewTransaction(tx)

	require.Error(t, err, "an transaction that failed pre-order checks was added to the pool")
	require.IsType(t, &transactionpool.ErrTransactionRejected{}, err, "error was not of the expected type")

	typedError := err.(*transactionpool.ErrTransactionRejected)
	require.Equal(t, expectedStatus, typedError.TransactionStatus, "error did not contain expected transaction status")

	require.NoError(t, h.verifyMocks(), "mocks were not called as expected")

}

func TestDoesNotAddTheSameTransactionTwice(t *testing.T) {
	h := NewHarness()

	tx := builders.TransferTransaction().Build()
	h.ignoringForwardMessages()

	h.addNewTransaction(tx)
	require.Error(t, h.addNewTransaction(tx), "a transaction was added twice to the pool")
}

func TestReturnsReceiptForTransactionThatHasAlreadyBeenCommitted(t *testing.T) {
	h := NewHarness()

	tx := builders.TransferTransaction().Build()
	h.ignoringForwardMessages()

	h.addNewTransaction(tx)
	h.reportTransactionAsCommitted(tx)

	receipt, err := h.txpool.AddNewTransaction(&services.AddNewTransactionInput{
		SignedTransaction: tx,
	})

	require.NoError(t, err, "a committed transaction that was added again was wrongly rejected")
	require.Equal(t, protocol.TRANSACTION_STATUS_COMMITTED, receipt.TransactionStatus, "expected transaction status to be committed")
	require.Equal(t, hash.CalcSha256(tx.Raw()), receipt.TransactionReceipt.Txhash(), "expected transaction receipt to contain transaction hash")
}
