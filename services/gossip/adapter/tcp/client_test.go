// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
	"time"
)

func TestClientConnection_EnablesQueueWhenConnectedToServer_AndDisablesQueueOnDisconnect(t *testing.T) {
	with.Context(func(ctx context.Context) {
		server := newServerStub(t)
		with.Logging(t, func(parent *with.LoggingHarness) {

			defer server.Close()

			client := server.createClientAndConnect(ctx, t, parent.Logger, 20*time.Hour) // so that we don't send keep alives

			waitForQueueEnabled(t, client)

			client.disconnect()

			waitForQueueDisabled(t, client)

			require.Zero(t, server.readSomeBytes(), "client shouldn't have sent anything")
		})
	})

}

func TestClientConnection_ReconnectsWhenServerDisconnects_AndSendKeepAlive(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			server := newServerStub(t)
			defer server.Close()

			client := server.createClientAndConnect(ctx, t, parent.Logger, 10*time.Millisecond)
			waitForQueueEnabled(t, client)

			server.forceDisconnect(t)

			server.acceptClientConnection(t)
			waitForQueueEnabled(t, client)

			require.NotZero(t, server.readSomeBytes(), "client didn't send keep alive")
			require.NotZero(t, server.readSomeBytes(), "client didn't send second keep alive")
		})
	})
}

type timeouts struct {
	keepAliveInterval time.Duration
}

func (t *timeouts) GossipNetworkTimeout() time.Duration {
	return TEST_NETWORK_TIMEOUT
}

func (t *timeouts) GossipReconnectInterval() time.Duration {
	return 100 * time.Millisecond
}

func (t *timeouts) GossipConnectionKeepAliveInterval() time.Duration {
	return t.keepAliveInterval
}

type serverStub struct {
	listener net.Listener
	conn     net.Conn
	port     int
}

func newServerStub(t testing.TB) *serverStub {
	s := &serverStub{}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "test peer server could not listen")
	s.listener = listener
	s.port = listener.Addr().(*net.TCPAddr).Port
	return s
}

func (s *serverStub) Close() {
	_ = s.conn.Close()
	_ = s.listener.Close()
}

func (s *serverStub) acceptClientConnection(t testing.TB) {
	conn, err := s.listener.Accept()
	require.NoError(t, err, "test peer server could not accept connection")
	_ = conn.SetReadDeadline(time.Now().Add(HARNESS_PEER_READ_TIMEOUT))
	s.conn = conn
}

func (s *serverStub) readSomeBytes() int {
	bytesRead, _ := s.conn.Read(make([]byte, 4))
	return bytesRead
}

func (s *serverStub) createClientAndConnect(ctx context.Context, t testing.TB, logger log.Logger, keepAliveInterval time.Duration) *clientConnection {
	registry := metric.NewRegistry()
	peer := config.NewHardCodedGossipPeer(s.port, "127.0.0.1", "012345")
	client := newClientConnection(peer, logger, registry, getMetrics(registry), &timeouts{keepAliveInterval: keepAliveInterval})
	client.connect(ctx)
	s.acceptClientConnection(t)
	return client
}

func (s *serverStub) forceDisconnect(t testing.TB) {
	require.NoError(t, s.conn.Close())
}

func waitForQueueEnabled(t *testing.T, client *clientConnection) {
	require.True(t, test.Eventually(HARNESS_OUTGOING_CONNECTIONS_INIT_TIMEOUT, func() bool {
		return !client.queue.disabled()
	}), "client did not connect to server within timeout")
}

func waitForQueueDisabled(t *testing.T, client *clientConnection) {
	require.True(t, test.Eventually(HARNESS_OUTGOING_CONNECTIONS_INIT_TIMEOUT, func() bool {
		return client.queue.disabled()
	}), "client did not disable queue on disconnect")
}
