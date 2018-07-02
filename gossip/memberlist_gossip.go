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
	IncomingMessages []string
	OutgoingMessages []string
}

func (d GossipDelegate) NodeMeta(limit int) []byte {
	return []byte{}
}

func (d GossipDelegate) NotifyMsg(message []byte) {
	fmt.Println("Message received", string(message))
	d.IncomingMessages = append(d.IncomingMessages, string(message))
}

func (d GossipDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	var result [][]byte

	fmt.Println("Outgoing messages", d.OutgoingMessages)

	for _, message := range d.OutgoingMessages {
		result = append(result, []byte(message))
	}

	return result

	// return [][]byte{}
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
	g.delegate.OutgoingMessages = append(g.delegate.OutgoingMessages, message)
}
