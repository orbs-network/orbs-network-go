package adapter

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/hashicorp/memberlist"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"time"
)

// TODO: move this to regular config model
type MemberlistGossipConfig struct {
	PublicKey primitives.Ed25519PublicKey
	Port      int
	Peers     []string
}

// TODO: this needs to be private but had to be this way because it exports Join in main
type MemberlistTransport struct {
	list       *memberlist.Memberlist
	listConfig *MemberlistGossipConfig
	delegate   *gossipDelegate
	listeners  map[string]TransportListener
}

type gossipDelegate struct {
	Name             string
	OutgoingMessages *memberlist.TransmitLimitedQueue
	parent           *MemberlistTransport
}

func (d gossipDelegate) NodeMeta(limit int) []byte {
	return []byte{}
}

func (d gossipDelegate) NotifyMsg(rawMessage []byte) {
	// No need to queue, we can dispatch right here
	payloads := decodeByteArray(rawMessage)
	fmt.Printf("Gossip: message received by %s, %v\n", d.Name, payloads)
	d.parent.receive(payloads)
}

func (d gossipDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	broadcasts := d.OutgoingMessages.GetBroadcasts(overhead, limit)
	if len(broadcasts) > 0 {
		fmt.Println("Outgoing messages", len(broadcasts))
	}
	return broadcasts
}

func (d gossipDelegate) LocalState(join bool) []byte {
	return []byte{}
}

func (d gossipDelegate) MergeRemoteState(buf []byte, join bool) {
}

func NewGossipDelegate(nodeName string) gossipDelegate {
	return gossipDelegate{Name: nodeName}
}

// memberlist require node names in their cluster
func memberlistNodeName(publicKey primitives.Ed25519PublicKey) string {
	return fmt.Sprintf("node-pkey-%s", publicKey)
}

func NewMemberlistTransport(config MemberlistGossipConfig) Transport {
	fmt.Println("Creating memberlist with config", config)
	nodeName := memberlistNodeName(config.PublicKey)
	listConfig := memberlist.DefaultLocalConfig()
	listConfig.BindPort = config.Port
	listConfig.AdvertisePort = config.Port
	listConfig.Name = nodeName
	listConfig.GossipNodes = 21

	delegate := NewGossipDelegate(nodeName)
	delegate.OutgoingMessages = &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return 21
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
		fmt.Println(nodeName, "failed to join the cluster: "+err.Error())
	} else {
		fmt.Println(nodeName, "connected to", n, "hosts")
	}
	t := MemberlistTransport{
		list:       list,
		listConfig: &config,
		delegate:   &delegate,
		listeners:  make(map[string]TransportListener),
	}
	// this is terrible and should be purged
	delegate.parent = &t
	go t.remainConnectedLoop()
	return &t
}

func (t *MemberlistTransport) remainConnectedLoop() {
	for {
		t.join()
		time.Sleep(3 * time.Second)
	}
}

func (t *MemberlistTransport) join() {
	if len(t.list.Members()) < 2 {
		fmt.Println("Node does not have any peers, trying to join the cluster...", t.listConfig.Peers)
		t.list.Join(t.listConfig.Peers)
	}
}

func (t *MemberlistTransport) PrintPeers() {
	// Ask for members of the cluster
	for _, member := range t.list.Members() {
		fmt.Printf("Member: %s %s\n", member.Name, member.Addr)
	}
}

func (t *MemberlistTransport) Send(data *TransportData) error {
	if data.RecipientMode != gossipmessages.RECIPIENT_LIST_MODE_BROADCAST {
		//FIXME once we will be able to lookup a node name, replace with t.list.SendReliable(): https://godoc.org/github.com/hashicorp/memberlist#Memberlist.SendReliable
		fmt.Println("WARNING: Gossip: should not broadast targeted messages to everyone")
	}
	rawMessage := encodeByteArray(data.Payloads)
	t.delegate.OutgoingMessages.QueueBroadcast(&broadcast{msg: rawMessage})
	// TODO: add proper error handling
	return nil
}

func (t *MemberlistTransport) receive(payloads [][]byte) {
	fmt.Println("Gossip: triggering listeners")
	for _, l := range t.listeners {
		l.OnTransportMessageReceived(payloads)
	}
}

func (t *MemberlistTransport) RegisterListener(listener TransportListener, listenerPublicKey primitives.Ed25519PublicKey) {
	t.listeners[string(listenerPublicKey)] = listener
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

func encodeByteArray(payloads [][]byte) []byte {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	enc.Encode(payloads)
	return buffer.Bytes()
}

func decodeByteArray(data []byte) (res [][]byte) {
	buffer := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buffer)
	dec.Decode(&res)
	return
}
