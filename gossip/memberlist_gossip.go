package gossip

import (
	"fmt"

	"github.com/hashicorp/memberlist"
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
}

type GossipDelegate struct {
	Name             string
	OutgoingMessages *memberlist.TransmitLimitedQueue
}

func (d GossipDelegate) NodeMeta(limit int) []byte {
	return []byte{}
}

func (d GossipDelegate) NotifyMsg(message []byte) {
	fmt.Println("Message received", string(message))
	// No need to queue, we can dispatch right here
}

func (d GossipDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	broadcasts := d.OutgoingMessages.GetBroadcasts(overhead, limit)

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
