// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func TestDirectIncoming_ConnectionsAreListenedToWhileContextIsLive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h := newDirectHarnessWithConnectedPeers(t, ctx)

	connection, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", h.transport.serverPort))
	require.NoError(t, err, "test peer should be able connect to local transport")
	defer connection.Close()

	cancel()

	buffer := []byte{0}
	read, err := connection.Read(buffer)
	require.Equal(t, 0, read, "test peer should disconnect from local transport without reading anything")
	require.Error(t, err, "test peer should disconnect from local transport")

	eventuallyFailsConnecting := test.Eventually(test.EVENTUALLY_ADAPTER_TIMEOUT, func() bool {
		connection, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", h.transport.serverPort))
		if err != nil {
			return true
		} else {
			connection.Close()
			return false
		}
	})
	require.True(t, eventuallyFailsConnecting, "test peer should not be able to connect to local transport")
}

func TestDirectIncoming_TransportListenerReceivesData(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		h := newDirectHarnessWithConnectedPeers(t, ctx)
		defer h.cleanupConnectedPeers()

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
	test.WithContext(func(ctx context.Context) {

		h := newDirectHarnessWithConnectedPeers(t, ctx)
		defer h.cleanupConnectedPeers()

		h.expectTransportListenerNotCalled()

		buffer := exampleWireProtocolEncoding_Payloads_0x11_0x2233()
		written, err := h.peerTalkerConnection.Write(buffer)
		require.NoError(t, err, "test peer could not write to local transport")
		require.Equal(t, len(buffer), written)

		h.verifyTransportListenerNotCalled(t)
	})
}

func TestDirectIncoming_TransportListenerDoesNotReceiveCorruptData_NumPayloads(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		h := newDirectHarnessWithConnectedPeers(t, ctx)
		defer h.cleanupConnectedPeers()

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
	test.WithContext(func(ctx context.Context) {

		h := newDirectHarnessWithConnectedPeers(t, ctx)
		defer h.cleanupConnectedPeers()

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
	test.WithContext(func(ctx context.Context) {

		h := newDirectHarnessWithConnectedPeers(t, ctx)
		defer h.cleanupConnectedPeers()

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
	test.WithContext(func(ctx context.Context) {

		h := newDirectHarnessWithConnectedPeers(t, ctx)
		defer h.cleanupConnectedPeers()

		buffer := exampleWireProtocolEncoding_Payloads_0x11_0x2233()[:6] // only 6 out of 20 bytes transferred
		written, err := h.peerTalkerConnection.Write(buffer)
		require.NoError(t, err, "test peer could not write to local transport")
		require.Equal(t, len(buffer), written)

		buffer = []byte{0} // dummy buffer just to see when the connection closes
		_, err = h.peerTalkerConnection.Read(buffer)
		require.Error(t, err, "test peer should be disconnected from local transport")
	})
}
