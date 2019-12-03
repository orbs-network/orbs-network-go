// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	membuffers "github.com/orbs-network/membuffers/go"
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

func TestOutgoingConnection_EnablesQueueWhenConnectedToServer_AndDisablesQueueOnDisconnect(t *testing.T) {
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

func TestOutgoingConnection_ReconnectsWhenServerDisconnects_AndSendKeepAlive(t *testing.T) {
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

			<-client.disconnect()
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
	t        testing.TB
}

func newServerStub(t testing.TB) *serverStub {
	s := &serverStub{t: t}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "test peer server could not listen")
	s.listener = listener
	s.port = listener.Addr().(*net.TCPAddr).Port
	return s
}

// TODO this is based on transportServer.receiveTransportData, consider unifying
func (s *serverStub) readMessage(ctx context.Context) [][]byte {
	var res [][]byte

	// receive num payloads
	sizeBuffer, err := readTotal(ctx, s.conn, 4, time.Second)
	require.NoError(s.t, err)
	numPayloads := membuffers.GetUint32(sizeBuffer)

	for i := uint32(0); i < numPayloads; i++ {
		// receive payload size
		sizeBuffer, err := readTotal(ctx, s.conn, 4, time.Second)
		require.NoError(s.t, err)
		payloadSize := membuffers.GetUint32(sizeBuffer)

		// receive payload data
		payload, err := readTotal(ctx, s.conn, payloadSize, time.Second)
		require.NoError(s.t, err)
		res = append(res, payload)

		// receive padding
		paddingSize := calcPaddingSize(uint32(len(payload)))
		if paddingSize > 0 {
			_, err := readTotal(ctx, s.conn, paddingSize, time.Second)
			require.NoError(s.t, err)
		}
	}

	return res
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

func (s *serverStub) createClientAndConnect(ctx context.Context, t testing.TB, logger log.Logger, keepAliveInterval time.Duration) *outgoingConnection {
	registry := metric.NewRegistry()
	peer := config.NewHardCodedGossipPeer(s.port, "127.0.0.1", "012345")
	client := newOutgoingConnection(peer, logger, registry, createOutgoingConnectionMetrics(registry), &timeouts{keepAliveInterval: keepAliveInterval})
	client.connect(ctx)
	s.acceptClientConnection(t)
	return client
}

func (s *serverStub) forceDisconnect(t testing.TB) {
	require.NoError(t, s.conn.Close())
}

func waitForQueueEnabled(t *testing.T, client *outgoingConnection) {
	require.True(t, test.Eventually(HARNESS_OUTGOING_CONNECTIONS_INIT_TIMEOUT, func() bool {
		return !client.queue.disabled()
	}), "client did not connect to server within timeout")
}

func waitForQueueDisabled(t *testing.T, client *outgoingConnection) {
	require.True(t, test.Eventually(HARNESS_OUTGOING_CONNECTIONS_INIT_TIMEOUT, func() bool {
		return client.queue.disabled()
	}), "client did not disable queue on disconnect")
}
