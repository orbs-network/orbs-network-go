package tcp

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
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

func TestDirectOutgoing_ConnectionsToAllPeersOnInitWhileContextIsLive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	h := newDirectHarnessWithConnectedPeers(t, ctx)
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

		h := newDirectHarnessWithConnectedPeers(t, ctx)
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

		h := newDirectHarnessWithConnectedPeers(t, ctx)
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

func concatSlices(slices ...[]byte) []byte {
	var tmp []byte
	for _, s := range slices {
		tmp = append(tmp, s...)
	}
	return tmp
}

// encoded examples of the gossip wire protocol spec:
// https://github.com/orbs-network/orbs-spec/blob/master/encoding/gossip/membuffers-over-tcp.md

func exampleWireProtocolEncoding_Payloads_0x11_0x2233() []byte {
	// encoding payloads: [][]byte{{0x11}, {0x22, 0x33}}
	field_NumPayloads := []byte{0x02, 0x00, 0x00, 0x00}      // little endian
	field_FirstPayloadSize := []byte{0x01, 0x00, 0x00, 0x00} // little endian
	field_FirstPayloadData := []byte{0x11}
	field_FirstPayloadPadding := []byte{0x00, 0x00, 0x00}     // round payload data to 4 bytes
	field_SecondPayloadSize := []byte{0x02, 0x00, 0x00, 0x00} // little endian
	field_SecondPayloadData := []byte{0x22, 0x33}
	field_SecondPayloadPadding := []byte{0x00, 0x00} // round payload data to 4 bytes
	return concatSlices(field_NumPayloads, field_FirstPayloadSize, field_FirstPayloadData, field_FirstPayloadPadding, field_SecondPayloadSize, field_SecondPayloadData, field_SecondPayloadPadding)
}

func exampleWireProtocolEncoding_CorruptNumPayloads() []byte {
	field_NumPayloads := []byte{0x99, 0x99, 0x99, 0x99}      // corrupt value (too big)
	field_FirstPayloadSize := []byte{0x01, 0x00, 0x00, 0x00} // little endian
	field_FirstPayloadData := []byte{0x11}
	field_FirstPayloadPadding := []byte{0x00, 0x00, 0x00} // round payload data to 4 bytes
	return concatSlices(field_NumPayloads, field_FirstPayloadSize, field_FirstPayloadData, field_FirstPayloadPadding)
}

func exampleWireProtocolEncoding_CorruptPayloadSize() []byte {
	field_NumPayloads := []byte{0x01, 0x00, 0x00, 0x00}      // little endian
	field_FirstPayloadSize := []byte{0x99, 0x99, 0x99, 0x99} // corrupt value (too big)
	field_FirstPayloadData := []byte{0x11}
	field_FirstPayloadPadding := []byte{0x00, 0x00, 0x00} // round payload data to 4 bytes
	return concatSlices(field_NumPayloads, field_FirstPayloadSize, field_FirstPayloadData, field_FirstPayloadPadding)
}

func exampleWireProtocolEncoding_KeepAlive() []byte {
	// encoding payloads: [][]byte{} (this is how a keep alive looks like = zero payloads)
	field_NumPayloads := []byte{0x00, 0x00, 0x00, 0x00} // little endian
	return concatSlices(field_NumPayloads)
}
