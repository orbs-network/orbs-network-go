package gossip

import (
	"fmt"

	"github.com/hashicorp/memberlist"
	"github.com/orbs-network/orbs-network-go/types"
)

type MemberlistGossipConfig struct {
	Name  string
	Port  int
	Peers []string
}

type MemberlistGossip struct {
	list                     *memberlist.Memberlist
	listConfig               MemberlistGossipConfig
	transactionListeners     []TransactionListener
	consensusListeners       []ConsensusListener
	pausedForwards           bool
	pendingTransactions      []types.Transaction
	failNextConsensusRequest bool
}

func (g *MemberlistGossip) RegisterTransactionListener(listener TransactionListener) {
	g.transactionListeners = append(g.transactionListeners, listener)
}

func (g *MemberlistGossip) RegisterConsensusListener(listener ConsensusListener) {
	g.consensusListeners = append(g.consensusListeners, listener)
}

func (g *MemberlistGossip) CommitTransaction(transaction *types.Transaction) {
	for _, l := range g.consensusListeners {
		l.OnCommitTransaction(transaction)
	}
}

func (g *MemberlistGossip) ForwardTransaction(transaction *types.Transaction) {
	if g.pausedForwards {
		g.pendingTransactions = append(g.pendingTransactions, *transaction)
	} else {
		g.forwardToAllListeners(transaction)
	}
}

func (g *MemberlistGossip) forwardToAllListeners(transaction *types.Transaction) {
	for _, l := range g.transactionListeners {
		l.OnForwardTransaction(transaction)
	}
}

func (g *MemberlistGossip) PauseForwards() {
	g.pausedForwards = true
}

func (g *MemberlistGossip) ResumeForwards() {
	g.pausedForwards = false
	for _, pendingTransaction := range g.pendingTransactions {
		g.forwardToAllListeners(&pendingTransaction)
	}
	g.pendingTransactions = nil
}

func (g *MemberlistGossip) FailConsensusRequests() {
	g.failNextConsensusRequest = true
}

func (g *MemberlistGossip) PassConsensusRequests() {
	g.failNextConsensusRequest = false
}

func (g *MemberlistGossip) HasConsensusFor(transaction *types.Transaction) (bool, error) {
	if g.failNextConsensusRequest {
		return true, &ErrGossipRequestFailed{}
	}

	for _, l := range g.consensusListeners {
		if !l.ValidateConsensusFor(transaction) {
			return false, nil
		}
	}
	return true, nil
}

func NewGossip(config MemberlistGossipConfig) *MemberlistGossip {
	fmt.Println("Creating memberlist with config", config)

	listConfig := memberlist.DefaultLocalConfig()
	listConfig.BindPort = config.Port
	listConfig.Name = config.Name

	list, err := memberlist.Create(listConfig)
	if err != nil {
		panic("Failed to create memberlist: " + err.Error())
	}

	// Join an existing cluster by specifying at least one known member.
	n, err := list.Join(config.Peers)

	if err != nil {
		fmt.Println("Failed to join cluster: " + err.Error())
	} else {
		fmt.Println("Connected to", n, "hosts")
	}

	returnObject := MemberlistGossip{}
	returnObject.list = list
	returnObject.listConfig = config

	return &returnObject
}

func (g *MemberlistGossip) Join() {
	if len(g.list.Members()) < 2 {
		fmt.Println("Node does not have any peers, trying to join the cluster...", g.listConfig.Peers)
		g.list.Join(g.listConfig.Peers)
	}
}

func (g *MemberlistGossip) PrintPeers() {
	// Ask for members of the cluster
	for _, member := range g.list.Members() {
		fmt.Printf("Member: %s %s\n", member.Name, member.Addr)
	}
}
