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

	listeners map[string]MessageReceivedListener
}

type GossipDelegate struct {
	Name             string
	OutgoingMessages *memberlist.TransmitLimitedQueue
	parent           *MemberlistGossip
}

func (d GossipDelegate) NodeMeta(limit int) []byte {
	return []byte{}
}

func (d GossipDelegate) NotifyMsg(rawMessage []byte) {
	fmt.Println("Message received", string(rawMessage))
	// No need to queue, we can dispatch right here

	message := Message{}
	err := json.Unmarshal(rawMessage, &message)

	if err != nil {
		fmt.Println("Failed to unmarshal message", err)
	}

	fmt.Println("Unmarshalled message as", message)

	d.parent.receive(message)
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

func NewMemberlistTransport(config MemberlistGossipConfig) *MemberlistGossip {
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

	returnObject := MemberlistGossip{
		list:       list,
		listConfig: &config,
		delegate:   &delegate,
		listeners:  make(map[string]MessageReceivedListener),
	}

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

func (g *MemberlistGossip) Broadcast(message *Message) error {
	jsonValue, _ := json.Marshal(message)

	g.delegate.OutgoingMessages.QueueBroadcast(&broadcast{msg: jsonValue})
	g.receive(Message{message.Sender, message.Type, message.Payload})

	// add proper error handling

	return nil
}

//TODO pause/resume unicasts as well as broadcasts
func (g *MemberlistGossip) Unicast(recipientId string, message *Message) error {
	fmt.Println("Gossip: Unicast not implemented")
	// go g.listeners[recipientId].OnMessageReceived(message)

	return nil
}

func (g *MemberlistGossip) receive(message Message) {
	fmt.Println("Gossip: triggering listeners")
	for _, l := range g.listeners {
		l.OnMessageReceived(&message)
	}
}

func (g *MemberlistGossip) RegisterListener(listener MessageReceivedListener, myNodeId string) {
	g.listeners[myNodeId] = listener
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
