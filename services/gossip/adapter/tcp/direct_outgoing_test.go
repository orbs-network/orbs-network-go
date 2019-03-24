// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDirectOutgoing_ConnectionsToAllPeersOnInitWhileContextIsLive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	h := newDirectHarnessWithConnectedPeersWithoutKeepAlives(t, ctx)
	defer h.cleanupConnectedPeers()

	cancel()

	for i := 0; i < NETWORK_SIZE-1; i++ {
		buffer := []byte{0}
		read, err := h.peersListenersConnections[i].Read(buffer)
		require.Equal(t, 0, read, "local transport should disconnect from test peer without reading anything")
		require.Error(t, err, "local transport should disconnect from test peer")
	}
}

func TestDirectOutgoing_ConnectionReconnectsOnFailure(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		h := newDirectHarnessWithConnectedPeers(t, ctx)
		defer h.cleanupConnectedPeers()

		for numForcefulDisconnect := 0; numForcefulDisconnect < 2; numForcefulDisconnect++ {
			err := h.reconnect(numForcefulDisconnect % NETWORK_SIZE)

			require.NoError(t, err, "test peer server could not accept connection from local transport")
		}
	})
}

func TestDirectOutgoing_AdapterSendsBroadcast(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		h := newDirectHarnessWithConnectedPeersWithoutKeepAlives(t, ctx)
		defer h.cleanupConnectedPeers()

		err := h.transport.Send(ctx, &adapter.TransportData{
			SenderNodeAddress:      h.config.NodeAddress(),
			RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
			RecipientNodeAddresses: nil,
			Payloads:               [][]byte{{0x11}, {0x22, 0x33}},
		})
		require.NoError(t, err, "adapter Send should not fail")

		for i := 0; i < NETWORK_SIZE-1; i++ {
			data, err := h.peerListenerReadTotal(i, 20)
			require.NoError(t, err, "test peer server could not read from local transport")
			require.Equal(t, exampleWireProtocolEncoding_Payloads_0x11_0x2233(), data)
		}
	})
}

func TestDirectOutgoing_AdapterSendsUnicast(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		h := newDirectHarnessWithConnectedPeersWithoutKeepAlives(t, ctx)
		defer h.cleanupConnectedPeers()

		err := h.transport.Send(ctx, &adapter.TransportData{
			SenderNodeAddress:      h.config.NodeAddress(),
			RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
			RecipientNodeAddresses: []primitives.NodeAddress{h.nodeAddressForPeer(1)},
			Payloads:               [][]byte{{0x11}, {0x22, 0x33}},
		})
		require.NoError(t, err, "adapter Send should not fail")

		data, err := h.peerListenerReadTotal(1, 20)
		require.NoError(t, err, "test peer server could not read from local transport")
		require.Equal(t, exampleWireProtocolEncoding_Payloads_0x11_0x2233(), data)
	})
}

func TestDirectOutgoing_SendsKeepAliveWhenNothingToSend(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		h := newDirectHarnessWithConnectedPeers(t, ctx)
		defer h.cleanupConnectedPeers()

		for numKeepAliveSent := 0; numKeepAliveSent < 2; numKeepAliveSent++ {
			data, err := h.peerListenerReadTotal(1, 4)
			require.NoError(t, err, "test peer server could not read keepalive from local transport")
			require.Equal(t, exampleWireProtocolEncoding_KeepAlive(), data)
		}
	})
}

// wanted to simulate Timeout on Send instead of Error but was unable to
func TestDirectOutgoing_ErrorDuringSendCausesReconnect(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		h := newDirectHarnessWithConnectedPeers(t, ctx)
		defer h.cleanupConnectedPeers()

		err := h.transport.Send(ctx, &adapter.TransportData{
			SenderNodeAddress:      h.config.NodeAddress(),
			RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
			RecipientNodeAddresses: []primitives.NodeAddress{h.nodeAddressForPeer(1)},
			Payloads:               [][]byte{{0x11}, {0x22, 0x33}},
		})
		require.NoError(t, err, "adapter Send should not fail")

		h.peersListenersConnections[1].Close() // break the pipe during Send

		h.peersListenersConnections[1], err = h.peersListeners[1].Accept()
		require.NoError(t, err, "test peer server did not accept new connection from local transport")

		data, err := h.peerListenerReadTotal(1, 4)
		require.NoError(t, err, "test peer server could not read keepalive from local transport")
		require.Equal(t, exampleWireProtocolEncoding_KeepAlive(), data)
	})
}
