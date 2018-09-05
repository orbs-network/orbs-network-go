package adapter

import (
	"context"
	"fmt"
	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/pkg/errors"
	"net"
	"sync"
	"time"
)

type Config interface {
	NodePublicKey() primitives.Ed25519PublicKey
	FederationNodes(asOfBlock uint64) map[string]config.FederationNode
	GossipConnectionKeepAliveInterval() time.Duration
}

type directTransport struct {
	config    Config
	reporting log.BasicLogger

	peerQueues map[string]chan *TransportData // does not require mutex to read

	mutex             *sync.RWMutex
	transportListener TransportListener
	serverReady       bool
}

func NewDirectTransport(ctx context.Context, config Config, reporting log.BasicLogger) Transport {
	t := &directTransport{
		config:    config,
		reporting: reporting.For(log.String("adapter", "gossip")),

		peerQueues: make(map[string]chan *TransportData),

		mutex: &sync.RWMutex{},
	}

	// client channels (not under mutex, before all goroutines)
	for peerNodeKey, peer := range t.config.FederationNodes(0) {
		if !peer.NodePublicKey().Equal(t.config.NodePublicKey()) {
			t.peerQueues[peerNodeKey] = make(chan *TransportData)
		}
	}

	// server goroutine
	go t.serverMainLoop(ctx, t.getListenPort())

	// client goroutines
	for peerNodeKey, peer := range t.config.FederationNodes(0) {
		if !peer.NodePublicKey().Equal(t.config.NodePublicKey()) {
			peerAddress := fmt.Sprintf("%s:%d", peer.GossipEndpoint(), peer.GossipPort())
			go t.clientMainLoop(ctx, peerAddress, t.peerQueues[peerNodeKey])
		}
	}

	return t
}

func (t *directTransport) RegisterListener(listener TransportListener, listenerPublicKey primitives.Ed25519PublicKey) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.transportListener = listener
}

func (t *directTransport) Send(data *TransportData) error {
	switch data.RecipientMode {
	case gossipmessages.RECIPIENT_LIST_MODE_BROADCAST:
		for _, peerQueue := range t.peerQueues {
			peerQueue <- data
		}
	case gossipmessages.RECIPIENT_LIST_MODE_LIST:
		for _, recipientPublicKey := range data.RecipientPublicKeys {
			if peerQueue, found := t.peerQueues[recipientPublicKey.KeyForMap()]; found {
				peerQueue <- data
			} else {
				return errors.Errorf("unknown recepient public key: %s", recipientPublicKey.String())
			}
		}
	case gossipmessages.RECIPIENT_LIST_MODE_ALL_BUT_LIST:
		panic("Not implemented")
	}
	return errors.Errorf("unknown recipient mode: %s", data.RecipientMode.String())
}

func (t *directTransport) getListenPort() uint16 {
	nodePublicKey := t.config.NodePublicKey()
	nodeConfig, found := t.config.FederationNodes(0)[nodePublicKey.KeyForMap()]
	if !found {
		err := errors.Errorf("fatal error - gossip configuration (port and endpoint) not found for my public key: %s", nodePublicKey.String())
		t.reporting.Error(err.Error())
		panic(err)
	}
	return nodeConfig.GossipPort()
}

func (t *directTransport) serverListenForIncomingConnections(ctx context.Context, listenPort uint16) (net.Listener, error) {
	// TODO: migrate to ListenConfig which has better support of contexts (go 1.11 required)
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		return nil, err
	}

	// this goroutine will shut down the server gracefully when context is done
	go func() {
		<-ctx.Done()
		t.mutex.Lock()
		defer t.mutex.Unlock()
		t.serverReady = false
		listener.Close()
	}()

	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.serverReady = true

	return listener, err
}

func (t *directTransport) isServerReady() bool {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.serverReady
}

func (t *directTransport) serverMainLoop(ctx context.Context, listenPort uint16) {
	listener, err := t.serverListenForIncomingConnections(ctx, listenPort)
	if err != nil {
		err = errors.Wrapf(err, "gossip transport cannot listen on port %d", listenPort)
		t.reporting.Error(err.Error())
		panic(err)
	}
	t.reporting.Info("gossip transport server listening", log.Uint32("port", uint32(listenPort)))

	for {
		conn, err := listener.Accept()
		if err != nil {
			if !t.isServerReady() {
				t.reporting.Info("incoming connection accept stopped since server is shutting down")
				return
			}
			t.reporting.Info("incoming connection accept error", log.Error(err))
			continue
		}
		go t.serverHandleIncomingConnection(ctx, conn)
	}
}

func (t *directTransport) serverHandleIncomingConnection(ctx context.Context, conn net.Conn) {
	t.reporting.Info("successful incoming gossip transport connection", log.String("peer", conn.RemoteAddr().String()))
	// TODO: add a whitelist for IPs we're willing to accept connections from

	<-ctx.Done() // TODO: replace with actual send/receive logic
	conn.Close()
}

func (t *directTransport) clientMainLoop(ctx context.Context, address string, msgs chan *TransportData) {
	for {

		t.reporting.Info("attempting outgoing transport connection", log.String("server", address))
		conn, err := net.Dial("tcp", address)

		if err != nil {
			t.reporting.Info("cannot connect to gossip peer endpoint", log.Error(err))
			time.Sleep(t.config.GossipConnectionKeepAliveInterval())
			continue
		}

		if !t.clientHandleOutgoingConnection(ctx, conn, msgs) {
			return
		}

	}
}

// returns true if should attempt reconnect on error
func (t *directTransport) clientHandleOutgoingConnection(ctx context.Context, conn net.Conn, msgs chan *TransportData) bool {
	t.reporting.Info("successful outgoing gossip transport connection", log.String("peer", conn.RemoteAddr().String()))

	for {
		select {
		case data := <-msgs:
			err := t.sendTransportData(ctx, conn, data)
			if err != nil {
				t.reporting.Info("failed sending transport data, reconnecting", log.Error(err), log.String("peer", conn.RemoteAddr().String()))
				conn.Close()
				return true
			}
		case <-time.After(t.config.GossipConnectionKeepAliveInterval()):
			err := t.sendKeepAlive(ctx, conn)
			if err != nil {
				t.reporting.Info("failed sending keepalive, reconnecting", log.Error(err), log.String("peer", conn.RemoteAddr().String()))
				conn.Close()
				return true
			}
		case <-ctx.Done():
			t.reporting.Info("client loop stopped since server is shutting down")
			conn.Close()
			return false
		}
	}
}

func (t *directTransport) sendTransportData(ctx context.Context, conn net.Conn, data *TransportData) error {
	t.reporting.Info("sending transport data", log.Int("payloads", len(data.Payloads)), log.String("peer", conn.RemoteAddr().String()))

	zeroBuffer := make([]byte, 4)
	sizeBuffer := make([]byte, 4)

	// send num payloads
	membuffers.WriteUint32(sizeBuffer, uint32(len(data.Payloads)))
	written, _ := conn.Write(sizeBuffer)
	if written != 4 {
		return errors.Errorf("attempted to write %d bytes but only wrote %d", 4, written)
	}

	for _, payload := range data.Payloads {
		// send payload size
		membuffers.WriteUint32(sizeBuffer, uint32(len(payload)))
		written, _ := conn.Write(sizeBuffer)
		if written != 4 {
			return errors.Errorf("attempted to write %d bytes but only wrote %d", 4, written)
		}

		// send payload data
		written, _ = conn.Write(payload)
		if written != len(payload) {
			return errors.Errorf("attempted to write %d bytes but only wrote %d", len(payload), written)
		}

		// send padding
		paddingSize := calcPaddingSize(len(payload))
		if paddingSize > 0 {
			written, _ = conn.Write(zeroBuffer[:paddingSize])
			if written != paddingSize {
				return errors.Errorf("attempted to write %d bytes but only wrote %d", paddingSize, written)
			}
		}
	}

	return nil
}

func calcPaddingSize(size int) int {
	const contentAlignment = 4
	alignedSize := (size + contentAlignment - 1) / contentAlignment * contentAlignment
	return alignedSize - size
}

func (t *directTransport) sendKeepAlive(ctx context.Context, conn net.Conn) error {
	t.reporting.Info("sending keepalive", log.String("peer", conn.RemoteAddr().String()))

	zeroBuffer := make([]byte, 4)

	// send zero num payloads
	written, _ := conn.Write(zeroBuffer)
	if written != 4 {
		return errors.Errorf("attempted to write %d bytes but only wrote %d", 4, written)
	}

	return nil
}
