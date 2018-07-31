package test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/transactionpool"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

var _ = Describe("transaction pool", func() {

	var (
		gossip  *gossiptopics.MockTransactionRelay
		service services.TransactionPool
	)

	BeforeEach(func() {
		log := instrumentation.GetLogger()
		gossip = &gossiptopics.MockTransactionRelay{}
		gossip.When("RegisterTransactionRelayHandler", mock.Any).Return()
		service = transactionpool.NewTransactionPool(gossip, log)
	})

	It("forwards a new valid transaction with gossip", func() {

		tx := builders.TransferTransaction().Build()

		gossip.When("BroadcastForwardedTransactions", &gossiptopics.ForwardedTransactionsInput{
			Message: &gossipmessages.ForwardedTransactionsMessage{
				SignedTransactions: []*protocol.SignedTransaction{tx},
			},
		}).Return(&gossiptopics.EmptyOutput{}, nil).Times(1)

		_, err := service.AddNewTransaction(&services.AddNewTransactionInput{
			SignedTransaction: tx,
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(gossip).To(test.ExecuteAsPlanned())

	})

	It("does not forward an invalid transaction with gossip", func() {

		tx := builders.TransferTransaction().WithInvalidContent().Build()

		gossip.When("BroadcastForwardedTransactions", mock.Any).Return(&gossiptopics.EmptyOutput{}, nil).Times(0)

		_, err := service.AddNewTransaction(&services.AddNewTransactionInput{
			SignedTransaction: tx,
		})

		Expect(err).To(HaveOccurred())
		Expect(gossip).To(test.ExecuteAsPlanned())

	})

})
