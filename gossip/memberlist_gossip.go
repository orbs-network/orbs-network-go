package gossip

import (
	"encoding/json"
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
	list       *memberlist.Memberlist
	listConfig *MemberlistGossipConfig
	delegate   *GossipDelegate

	transactionListeners []TransactionListener
	consensusListeners   []ConsensusListener
	pendingTransactions  []types.Transaction
}

type GossipDelegate struct {
	Name             string
	OutgoingMessages *memberlist.TransmitLimitedQueue
	parent           *MemberlistGossip
}

func (d GossipDelegate) NodeMeta(limit int) []byte {
	return []byte{}
}

func (d GossipDelegate) NotifyMsg(message []byte) {
	fmt.Println("Message received", string(message))
	// No need to queue, we can dispatch right here

	var jsonValue interface{}
	err := json.Unmarshal(message, &jsonValue)

	m := jsonValue.(map[string]interface{})

	if err == nil {
		switch m["type"] {
		case "CommitTransaction":
			fmt.Println("TX", m["payload"])

			txContainer := m["payload"].(map[string]interface{})

			tx := &types.Transaction{
				Value:   int(txContainer["Value"].(float64)),
				Invalid: txContainer["Invalid"].(bool),
			}

			for _, l := range d.parent.consensusListeners {
				l.OnCommitTransaction(tx)
			}
		}
	}
	fmt.Println("Unmarshalled json", jsonValue)
}

func (d GossipDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	broadcasts := d.OutgoingMessages.GetBroadcasts(overhead, limit)

	if len(broadcasts) > 0 {
		fmt.Println("Outgoing messages")
	}

	for _, message := range broadcasts {
		fmt.Println(string(message))
	}

	return broadcasts
}

func (d GossipDelegate) LocalState(join bool) []byte {
	return []byte{}
}

func (d GossipDelegate) MergeRemoteState(buf []byte, join bool) {

}

func NewGossipDelegate(nodeName string) GossipDelegate {
	return GossipDelegate{Name: nodeName}
}

func NewGossip(config MemberlistGossipConfig) *MemberlistGossip {
	fmt.Println("Creating memberlist with config", config)

	listConfig := memberlist.DefaultLocalConfig()
	listConfig.BindPort = config.Port
	listConfig.Name = config.Name

	delegate := NewGossipDelegate(config.Name)
	delegate.OutgoingMessages = &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return len(config.Peers) - 1
		},
		RetransmitMult: listConfig.RetransmitMult,
	}

	listConfig.Delegate = &delegate

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
	returnObject.listConfig = &config
	returnObject.delegate = &delegate

	// this is terrible and should be purged
	delegate.parent = &returnObject

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

func (g *MemberlistGossip) SendMessage(message string) {
	fmt.Println("Sending a message", message)
	g.delegate.OutgoingMessages.QueueBroadcast(&broadcast{msg: []byte(message)})
}

func (g *MemberlistGossip) ForwardTransaction(transaction *types.Transaction) {
	fmt.Println("ForwardTransaction is not implemented")
}

func (g *MemberlistGossip) CommitTransaction(transaction *types.Transaction) {
	fmt.Println("Committing transaction")
	for _, l := range g.consensusListeners {
		l.OnCommitTransaction(transaction)
	}

	wrapper := map[string]interface{}{
		"type":    "CommitTransaction",
		"payload": transaction,
	}

	jsonValue, _ := json.Marshal(wrapper)
	g.SendMessage(string(jsonValue))
}

func (g *MemberlistGossip) HasConsensusFor(transaction *types.Transaction) (bool, error) {
	fmt.Println("Checking consensus for transaction", transaction)

	for _, l := range g.consensusListeners {
		if !l.ValidateConsensusFor(transaction) {
			return false, nil
		}
	}
	return true, nil
}

func (g *MemberlistGossip) RegisterTransactionListener(listener TransactionListener) {
	fmt.Println("Registering transaction listener")
	g.transactionListeners = append(g.transactionListeners, listener)
}

func (g *MemberlistGossip) RegisterConsensusListener(listener ConsensusListener) {
	fmt.Println("Registering consensus listener")
	g.consensusListeners = append(g.consensusListeners, listener)
}

type broadcast struct {
	msg    []byte
	notify chan<- struct{}
}

func (b *broadcast) Invalidates(other memberlist.Broadcast) bool {
	return false
}

func (b *broadcast) Message() []byte {
	return b.msg
}

func (b *broadcast) Finished() {
	if b.notify != nil {
		close(b.notify)
	}
}
