package test

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/services/transactionpool"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
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

func (h *harness) addNewTransaction(tx *protocol.SignedTransaction) (*services.AddNewTransactionOutput, error) {
	out, err := h.txpool.AddNewTransaction(&services.AddNewTransactionInput{
		SignedTransaction: tx,
	})

	return out, err
}

func (h *harness) reportTransactionAsCommitted(transaction *protocol.SignedTransaction) {
	h.txpool.CommitTransactionReceipts(&services.CommitTransactionReceiptsInput{
		LastCommittedBlockHeight: 1,
		TransactionReceipts: []*protocol.TransactionReceipt{
			(&protocol.TransactionReceiptBuilder{
				Txhash: digest.CalcTxHash(transaction.Transaction()),
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

func (h *harness) handleForwardFrom(tx *protocol.SignedTransaction, sender primitives.Ed25519PublicKey) {
	h.txpool.HandleForwardedTransactions(&gossiptopics.ForwardedTransactionsInput{
		Message: &gossipmessages.ForwardedTransactionsMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{SenderPublicKey: sender}).Build(),
			SignedTransactions: []*protocol.SignedTransaction{tx},
		},
	})
}

func NewHarness() *harness {
	gossip := &gossiptopics.MockTransactionRelay{}
	gossip.When("RegisterTransactionRelayHandler", mock.Any).Return()

	virtualMachine := &services.MockVirtualMachine{}
	virtualMachine.When("TransactionSetPreOrder", mock.Any).Return(&services.TransactionSetPreOrderOutput{PreOrderResults: []protocol.TransactionStatus{protocol.TRANSACTION_STATUS_PENDING}})

	service := transactionpool.NewTransactionPool(gossip, virtualMachine, instrumentation.GetLogger())

	return &harness{txpool: service, gossip: gossip, vm: virtualMachine}
}
