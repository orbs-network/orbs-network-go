// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
	"time"
)

func TestDirectIncoming_ConnectionsAreListenedToWhileContextIsLive(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		h := newDirectHarnessWithConnectedPeers(t, ctx, parent)

		connection, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", h.transport.GetServerPort()))
		require.NoError(t, err, "test peer should be able connect to local transport")
		defer connection.Close()

		h.transport.GracefulShutdown(ctx)

		buffer := []byte{0}
		read, err := connection.Read(buffer)
		require.Equal(t, 0, read, "test peer should disconnect from local transport without reading anything")
		require.Error(t, err, "test peer should disconnect from local transport")

		eventuallyFailsConnecting := test.Eventually(test.EVENTUALLY_ADAPTER_TIMEOUT, func() bool {
			connection, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", h.transport.GetServerPort()))
			if err != nil {
				return true
			} else {
				connection.Close()
				return false
			}
		})
		require.True(t, eventuallyFailsConnecting, "test peer should not be able to connect to local transport")
	})
}

func TestDirectIncoming_TransportListenerReceivesData(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		h := newDirectHarnessWithConnectedPeers(t, ctx, parent)
		defer h.cleanupConnectedPeers()
		defer h.transport.GracefulShutdown(ctx)

		h.transport.RegisterListener(h.listenerMock, nil)
		h.expectTransportListenerCalled([][]byte{{0x11}, {0x22, 0x33}})

		buffer := exampleWireProtocolEncoding_Payloads_0x11_0x2233()
		written, err := h.peerTalkerConnection.Write(buffer)
		require.NoError(t, err, "test peer could not write to local transport")
		require.Equal(t, len(buffer), written)

		h.verifyTransportListenerCalled(t)
	})
}

func TestDirectIncoming_ReceivesDataWithoutListener(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		h := newDirectHarnessWithConnectedPeers(t, ctx, parent)
		defer h.cleanupConnectedPeers()
		defer h.transport.GracefulShutdown(ctx)

		h.expectTransportListenerNotCalled()

		buffer := exampleWireProtocolEncoding_Payloads_0x11_0x2233()
		written, err := h.peerTalkerConnection.Write(buffer)
		require.NoError(t, err, "test peer could not write to local transport")
		require.Equal(t, len(buffer), written)

		h.verifyTransportListenerNotCalled(t)
	})
}

func TestDirectIncoming_TransportListenerDoesNotReceiveCorruptData_NumPayloads(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		h := newDirectHarnessWithConnectedPeers(t, ctx, parent)
		defer h.cleanupConnectedPeers()
		defer h.transport.GracefulShutdown(ctx)

		h.transport.RegisterListener(h.listenerMock, nil)
		h.expectTransportListenerNotCalled()

		buffer := exampleWireProtocolEncoding_CorruptNumPayloads()
		written, err := h.peerTalkerConnection.Write(buffer)
		require.NoError(t, err, "test peer could not write to local transport")
		require.Equal(t, len(buffer), written)

		buffer = []byte{0} // dummy buffer just to see when the connection closes
		_, err = h.peerTalkerConnection.Read(buffer)
		require.Error(t, err, "test peer should be disconnected from local transport")

		h.verifyTransportListenerNotCalled(t)
	})
}

func TestDirectIncoming_TransportListenerDoesNotReceiveCorruptData_PayloadSize(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		h := newDirectHarnessWithConnectedPeers(t, ctx, parent)
		defer h.cleanupConnectedPeers()
		defer h.transport.GracefulShutdown(ctx)

		h.transport.RegisterListener(h.listenerMock, nil)
		h.expectTransportListenerNotCalled()

		buffer := exampleWireProtocolEncoding_CorruptPayloadSize()
		written, err := h.peerTalkerConnection.Write(buffer)
		require.NoError(t, err, "test peer could not write to local transport")
		require.Equal(t, len(buffer), written)

		buffer = []byte{0} // dummy buffer just to see when the connection closes
		_, err = h.peerTalkerConnection.Read(buffer)
		require.Error(t, err, "test peer should be disconnected from local transport")

		h.verifyTransportListenerNotCalled(t)
	})
}

func TestDirectIncoming_TransportListenerIgnoresKeepAlives(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		h := newDirectHarnessWithConnectedPeers(t, ctx, parent)
		defer h.cleanupConnectedPeers()
		defer h.transport.GracefulShutdown(ctx)

		h.transport.RegisterListener(h.listenerMock, nil)
		h.expectTransportListenerCalled([][]byte{{0x11}, {0x22, 0x33}})

		for numKeepAliveReceived := 0; numKeepAliveReceived < 2; numKeepAliveReceived++ {
			buffer := exampleWireProtocolEncoding_KeepAlive()
			written, err := h.peerTalkerConnection.Write(buffer)
			require.NoError(t, err, "test peer could not write to local transport")
			require.Equal(t, len(buffer), written)
		}

		buffer := exampleWireProtocolEncoding_Payloads_0x11_0x2233()
		written, err := h.peerTalkerConnection.Write(buffer)
		require.NoError(t, err, "test peer could not write to local transport")
		require.Equal(t, len(buffer), written)

		h.verifyTransportListenerCalled(t)
	})
}

func TestDirectIncoming_TimeoutDuringReceiveCausesDisconnect(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {

		h := newDirectHarnessWithConnectedPeers(t, ctx, parent)
		defer h.cleanupConnectedPeers()
		defer h.transport.GracefulShutdown(ctx)

		buffer := exampleWireProtocolEncoding_Payloads_0x11_0x2233()[:6] // only 6 out of 20 bytes transferred
		written, err := h.peerTalkerConnection.Write(buffer)
		require.NoError(t, err, "test peer could not write to local transport")
		require.Equal(t, len(buffer), written)

		buffer = []byte{0} // dummy buffer just to see when the connection closes
		_, err = h.peerTalkerConnection.Read(buffer)
		require.Error(t, err, "test peer should be disconnected from local transport")
	})
}

type serverCfg struct {
	port uint16
}

func (s *serverCfg) GossipListenPort() uint16 {
	return s.port
}

func (s *serverCfg) GossipNetworkTimeout() time.Duration {
	return 100 * time.Millisecond
}

func TestDirectServer_PanicsOnPortAlreadyInUse(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {

		l1, err := net.Listen("tcp", ":0")
		require.NoError(t, err, "failed listening to port")
		port := l1.Addr().(*net.TCPAddr).Port

		cfg := &serverCfg{
			port: uint16(port),
		}

		server := newServer(cfg, harness.Logger, metric.NewRegistry())
		harness.Supervise(server)

		require.Panics(t, func() {
			server.startSupervisedMainLoop(ctx)
		}, "should have panicked on port already in use")
	})
}

func TestDirectServer_ShutsDownWithinTimeout_WhenHasZeroConnections(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) { // not using ConcurrencyHarness on purpose so that this test can block until shutdown
		cfg := &serverCfg{}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		server := newServer(cfg, harness.Logger, metric.NewRegistry())
		server.startSupervisedMainLoop(ctx)

		require.True(t, test.Eventually(100*time.Millisecond, func() bool {
			return server.IsListening()
		}))

		server.GracefulShutdown(context.Background())

		server.WaitUntilShutdown(context.Background())
	})
}

func TestDirectServer_KeepsAcceptingNewConnections_AfterAllClientConnectionsAreClosed(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		cfg := &serverCfg{}

		server := newServer(cfg, harness.Logger, metric.NewRegistry())
		harness.Supervise(server)
		server.startSupervisedMainLoop(ctx)
		defer server.GracefulShutdown(context.Background())

		require.True(t, test.Eventually(100*time.Millisecond, func() bool {
			return server.IsListening()
		}))

		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", server.getPort()))
		require.NoError(t, err, "failed connecting to server")
		_ = conn.Close()

		time.Sleep(100 * time.Millisecond) //TODO sleep is wrong

		conn, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", server.getPort()))
		require.NoError(t, err, "failed connecting to server")
		_ = conn.Close()
	})
}

func TestDirectServer_DoesNotHang_WhenWaitingForConnectionsToClose_OnShutDown(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		cfg := &serverCfg{}

		server := newServer(cfg, harness.Logger, metric.NewRegistry())
		harness.Supervise(server)
		server.startSupervisedMainLoop(ctx)

		require.True(t, test.Eventually(100*time.Millisecond, func() bool {
			return server.IsListening()
		}))

		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", server.getPort()))
		require.NoError(t, err, "failed connecting to server")

		server.GracefulShutdown(ctx)

		_ = conn.Close()
	})
}
