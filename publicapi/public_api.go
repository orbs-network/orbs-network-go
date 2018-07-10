package publicapi

import (
	"github.com/orbs-network/orbs-network-go/ledger"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services/gossip"
)

type publicApi struct {
	txRelay         gossip.TransactionRelay
	transactionPool services.TransactionPool
	ledger          ledger.Ledger
	events          instrumentation.Reporting
	isLeader        bool
}

func NewPublicApi(txRelay gossip.TransactionRelay,
	transactionPool services.TransactionPool,
	ledger ledger.Ledger,
	events instrumentation.Reporting,
	isLeader bool) services.PublicApi {
	return &publicApi{
		txRelay:         txRelay,
		transactionPool: transactionPool,
		ledger:          ledger,
		events:          events,
		isLeader:        isLeader,
	}
}

func (p *publicApi) SendTransaction(input *services.SendTransactionInput) (*services.SendTransactionOutput, error) {
	p.events.Info("enter_send_transaction")
	defer p.events.Info("exit_send_transaction")
	//TODO leader should also propagate transactions to other nodes
	tx := input.ClientRequest.SignedTransaction()
	if p.isLeader {
		p.transactionPool.AddNewTransaction(&services.AddNewTransactionInput{tx})
	} else {
		p.txRelay.BroadcastForwardedTransactions(&gossip.ForwardedTransactionsInput{Transactions:[]*protocol.SignedTransaction{tx}})
	}

	output := &services.SendTransactionOutput{}

	return output, nil
}

func (p *publicApi) CallMethod(input *services.CallMethodInput) (*services.CallMethodOutput, error) {
	p.events.Info("enter_call_method")
	defer p.events.Info("exit_call_method")

	output := &services.CallMethodOutput{ClientResponse: (&client.CallMethodResponseBuilder{
		OutputArguments: []*protocol.MethodArgumentBuilder{
			{Name: "balance", Type: protocol.MethodArgumentTypeUint64, Uint64: uint64(p.ledger.GetState())},
		},
	}).Build()}

	return output, nil
}

func (p *publicApi) GetTransactionStatus(input *services.GetTransactionStatusInput) (*services.GetTransactionStatusOutput, error) {
	panic("Not implemented")
}

func (p *publicApi) HandleTransactionResults(input *handlers.HandleTransactionResultsInput) (*handlers.HandleTransactionResultsOutput, error) {
	panic("Not implemented")
}
