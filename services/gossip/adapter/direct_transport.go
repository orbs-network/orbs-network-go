package adapter

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
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

	mutex             *sync.RWMutex
	transportListener TransportListener
	serverReady       bool
	peerQueues        map[string]chan *TransportData
}

func NewDirectTransport(ctx context.Context, config Config, reporting log.BasicLogger) Transport {
	t := &directTransport{
		config:    config,
		reporting: reporting.For(log.String("adapter", "gossip")),

		mutex:      &sync.RWMutex{},
		peerQueues: make(map[string]chan *TransportData),
	}

	// server goroutine
	go t.serverMainLoop(ctx, t.getListenPort())

	// client goroutines
	t.mutex.Lock()
	defer t.mutex.Unlock()
	for peerNodeKey, peer := range t.config.FederationNodes(0) {
		if !peer.NodePublicKey().Equal(t.config.NodePublicKey()) {
			t.peerQueues[peerNodeKey] = make(chan *TransportData)
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
	panic("not implemented")
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

func (t *directTransport) clientHandleOutgoingConnection(ctx context.Context, conn net.Conn, msgs chan *TransportData) bool {
	t.reporting.Info("successful outgoing gossip transport connection", log.String("peer", conn.RemoteAddr().String()))

	for {
		select {
		case <-msgs:
		case <-time.After(t.config.GossipConnectionKeepAliveInterval()):
			err := t.sendKeepAlive(conn)
			if err != nil {
				t.reporting.Info("failed sending keepalive", log.Error(err), log.String("peer", conn.RemoteAddr().String()))
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

func (t *directTransport) sendKeepAlive(conn net.Conn) error {
	t.reporting.Info("sending keepalive", log.String("peer", conn.RemoteAddr().String()))

	// TODO: replace with actual send/receive logic
	buffer := []byte{0}
	conn.SetDeadline(time.Now().Add(t.config.GossipConnectionKeepAliveInterval()))
	_, err := conn.Read(buffer)
	if err != nil {
		return err
	}

	return nil
}
