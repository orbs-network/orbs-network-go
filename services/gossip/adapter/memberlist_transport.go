package adapter

import (
	"fmt"
	"github.com/hashicorp/memberlist"
	"encoding/gob"
	"bytes"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

type memberlistGossipConfig struct {
	Name  string
	Port  int
	Peers []string
}

type memberlistTransport struct {
	list       *memberlist.Memberlist
	listConfig *memberlistGossipConfig
	delegate   *gossipDelegate
	listeners  map[string]TransportListener
}

type gossipDelegate struct {
	Name             string
	OutgoingMessages *memberlist.TransmitLimitedQueue
	parent           *memberlistTransport
}

func (d gossipDelegate) NodeMeta(limit int) []byte {
	return []byte{}
}

func (d gossipDelegate) NotifyMsg(rawMessage []byte) {
	fmt.Println("Message received", string(rawMessage))
	// No need to queue, we can dispatch right here
	messageWithPayloads := decodeByteArray(rawMessage)
	message := gossipmessages.HeaderReader(messageWithPayloads[0])
	payloads := messageWithPayloads[1:]
	fmt.Println("Unmarshalled message as", message)
	d.parent.receive(message, payloads)
}

func (d gossipDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	broadcasts := d.OutgoingMessages.GetBroadcasts(overhead, limit)
	if len(broadcasts) > 0 {
		fmt.Println("Outgoing messages")
	}
	for _, message := range broadcasts {
		fmt.Println(string(message))
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

func NewMemberlistTransport(config memberlistGossipConfig) Transport {
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
	returnObject := memberlistTransport{
		list:       list,
		listConfig: &config,
		delegate:   &delegate,
		listeners:  make(map[string]TransportListener),
	}
	// this is terrible and should be purged
	delegate.parent = &returnObject
	return &returnObject
}

func (t *memberlistTransport) Join() {
	if len(t.list.Members()) < 2 {
		fmt.Println("Node does not have any peers, trying to join the cluster...", t.listConfig.Peers)
		t.list.Join(t.listConfig.Peers)
	}
}

func (t *memberlistTransport) PrintPeers() {
	// Ask for members of the cluster
	for _, member := range t.list.Members() {
		fmt.Printf("Member: %s %s\n", member.Name, member.Addr)
	}
}

func (t *memberlistTransport) Send(message *gossipmessages.Header, payloads [][]byte) error {
	data := encodeByteArray(append([][]byte{message.Raw()}, payloads...))
	t.delegate.OutgoingMessages.QueueBroadcast(&broadcast{msg: data})
	t.receive(message, payloads)
	// TODO: add proper error handling
	return nil
}

func (t *memberlistTransport) receive(message *gossipmessages.Header, payloads [][]byte) {
	fmt.Println("Gossip: triggering listeners")
	for _, l := range t.listeners {
		l.OnTransportMessageReceived(message, payloads)
	}
}

func (t *memberlistTransport) RegisterListener(listener TransportListener, myNodeId string) {
	t.listeners[myNodeId] = listener
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
	var buffer bytes.Buffer
	dec := gob.NewDecoder(&buffer)
	dec.Decode(&res)
	return
}