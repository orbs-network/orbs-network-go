package publicapi

import (
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/types"
	"github.com/orbs-network/orbs-network-go/transactionpool"
	"github.com/orbs-network/orbs-network-go/ledger"
	"github.com/orbs-network/orbs-network-go/events"
)

type PublicApi interface {
	SendTransaction(transaction *types.Transaction)
	CallMethod() int
}

type publicApi struct {
	gossip          gossip.Gossip
	transactionPool transactionpool.TransactionPool
	ledger          ledger.Ledger
	events          events.Events
	isLeader        bool
}

func NewPublicApi(gossip gossip.Gossip,
	transactionPool transactionpool.TransactionPool,
	ledger ledger.Ledger,
	events events.Events,
	isLeader bool) PublicApi {
	return &publicApi{
		gossip: gossip,
		transactionPool: transactionPool,
		ledger: ledger,
		events: events,
		isLeader: isLeader,
	}
}


func (p *publicApi) SendTransaction(transaction *types.Transaction) {
	p.events.Report("enter_send_transaction")
	defer p.events.Report("exit_send_transaction")
	//TODO leader should also propagate transactions to other nodes
	if p.isLeader {
		p.transactionPool.Add(transaction)
	} else {
		p.gossip.ForwardTransaction(transaction)
	}
}

func (p *publicApi) CallMethod() int {
	p.events.Report("enter_call_method")
	defer p.events.Report("exit_call_method")
	return p.ledger.GetState()
}
