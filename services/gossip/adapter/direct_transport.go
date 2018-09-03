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
)

type Config interface {
	NodePublicKey() primitives.Ed25519PublicKey
	FederationNodes(asOfBlock uint64) map[string]config.FederationNode
}

type directTransport struct {
	config    Config
	reporting log.BasicLogger

	mutex             *sync.RWMutex
	transportListener TransportListener
	serverReady       bool
}

func NewDirectTransport(ctx context.Context, config Config, reporting log.BasicLogger) Transport {
	t := &directTransport{
		config:    config,
		reporting: reporting.For(log.String("adapter", "gossip")),

		mutex: &sync.RWMutex{},
	}

	go t.serverMainLoop(ctx, t.getListenPort())

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

func (t *directTransport) serverMainLoop(ctx context.Context, listenPort uint16) {
	listener, err := t.listenForIncomingConnections(ctx, listenPort)
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
		go t.handleIncomingConnection(conn)
	}
}

func (t *directTransport) listenForIncomingConnections(ctx context.Context, listenPort uint16) (net.Listener, error) {
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

func (t *directTransport) handleIncomingConnection(conn net.Conn) {
	t.reporting.Info("incoming gossip transport connection", log.String("client", conn.RemoteAddr().String()))
}
