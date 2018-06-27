package publicapi

import (
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/types"
	"github.com/orbs-network/orbs-network-go/transactionpool"
	"github.com/orbs-network/orbs-network-go/ledger"
)

type PublicApi interface {
	SendTransaction(transaction *types.Transaction)
	CallMethod() int
}

type publicApi struct {
	gossip          gossip.Gossip
	transactionPool transactionpool.TransactionPool
	ledger          ledger.Ledger
	isLeader        bool
}

func NewPublicApi(gossip gossip.Gossip, transactionPool transactionpool.TransactionPool, ledger ledger.Ledger, isLeader bool) PublicApi {
	return &publicApi{
		gossip: gossip,
		transactionPool: transactionPool,
		ledger: ledger,
		isLeader: isLeader,
	}
}


func (p *publicApi) SendTransaction(transaction *types.Transaction) {
	//TODO leader should also propagate transactions to other nodes
	if p.isLeader {
		p.transactionPool.Add(transaction)
	} else {
		p.gossip.ForwardTransaction(transaction)
	}
}

func (p *publicApi) CallMethod() int {
	return p.ledger.GetState()
}
