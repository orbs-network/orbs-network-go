package adapter

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func TestDirectIncoming_ConnectionsAreListenedToWhileContextIsLive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h := newDirectHarness().start(ctx)

	connection, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", h.myPort))
	defer connection.Close()
	require.NoError(t, err, "test peer should be able connect to local transport")

	cancel()

	buffer := []byte{0}
	read, err := connection.Read(buffer)
	require.Equal(t, 0, read, "test peer should disconnect from local transport without reading anything")
	require.Error(t, err, "test peer should disconnect from local transport")

	_, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", h.myPort))
	require.Error(t, err, "test peer should be able to connect to local transport")
}

func TestDirectOutgoing_ConnectionsToAllPeersOnInitWhileContextIsLive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	h := newDirectHarnessWithConnectedPeers(t, ctx)
	defer h.cleanupConnectedPeers()

	cancel()

	for i := 0; i < networkSize-1; i++ {
		buffer := []byte{0}
		read, err := h.peersListenersConnections[i].Read(buffer)
		require.Equal(t, 0, read, "local transport should disconnect from test peer without reading anything")
		require.Error(t, err, "local transport should disconnect from test peer")
	}
}

func TestDirectOutgoing_ConnectionReconnectsOnFailure(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		h := newDirectHarness().start(ctx)

		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", h.portForPeer(0)))
		defer listener.Close()
		require.NoError(t, err, "test peer server could not listen")

		connection, err := listener.Accept()
		defer connection.Close()
		require.NoError(t, err, "test peer server could not accept connection from local transport")

		for numForcefulDisconnect := 0; numForcefulDisconnect < 2; numForcefulDisconnect++ {
			connection.Close() // disconnect local transport forcefully

			connection, err = listener.Accept()
			require.NoError(t, err, "test peer server could not accept connection from local transport")
		}
	})
}

func TestDirectOutgoing_AdapterSendsBroadcast(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		h := newDirectHarnessWithConnectedPeers(t, ctx)
		defer h.cleanupConnectedPeers()

		h.transport.Send(&TransportData{
			SenderPublicKey:     h.config.NodePublicKey(),
			RecipientMode:       gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
			RecipientPublicKeys: nil,
			Payloads:            [][]byte{{0x11}, {0x22, 0x33}},
		})

		for i := 0; i < networkSize-1; i++ {
			data, err := h.peerListenerReadTotal(i, 20)
			require.NoError(t, err, "test peer server could not read from local transport")
			require.Equal(t, []byte{0x02, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x11, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x22, 0x33, 0x00, 0x00}, data)
		}
	})
}

func TestDirectOutgoing_AdapterSendsUnicast(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		h := newDirectHarnessWithConnectedPeers(t, ctx)
		defer h.cleanupConnectedPeers()

		h.transport.Send(&TransportData{
			SenderPublicKey:     h.config.NodePublicKey(),
			RecipientMode:       gossipmessages.RECIPIENT_LIST_MODE_LIST,
			RecipientPublicKeys: []primitives.Ed25519PublicKey{h.publicKeyForPeer(1)},
			Payloads:            [][]byte{{0x11}, {0x22, 0x33}},
		})

		data, err := h.peerListenerReadTotal(1, 20)
		require.NoError(t, err, "test peer server could not read from local transport")
		require.Equal(t, []byte{0x02, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x11, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x22, 0x33, 0x00, 0x00}, data)
	})
}

func TestDirectIncoming_TransportListenerReceivesData(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		h := newDirectHarnessWithConnectedPeers(t, ctx)
		defer h.cleanupConnectedPeers()

		h.transport.RegisterListener(h.listenerMock, nil)
		h.expectTransportListenerCalled([][]byte{{0x11}, {0x22, 0x33}})

		buffer := []byte{0x02, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x11, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x22, 0x33, 0x00, 0x00}
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

		buffer := []byte{0x02, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x11, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x22, 0x33, 0x00, 0x00}
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

		buffer := []byte{0x99, 0x99, 0x99, 0x99, 0x01, 0x00, 0x00, 0x00, 0x11, 0x00, 0x00, 0x00}
		written, err := h.peerTalkerConnection.Write(buffer)
		require.NoError(t, err, "test peer could not write to local transport")
		require.Equal(t, len(buffer), written)

		buffer = []byte{0}
		read, err := h.peerTalkerConnection.Read(buffer)
		require.Equal(t, 0, read, "test peer should be disconnected from local transport without reading anything")
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

		buffer := []byte{0x01, 0x00, 0x00, 0x00, 0x99, 0x99, 0x99, 0x99, 0x11, 0x00, 0x00, 0x00}
		written, err := h.peerTalkerConnection.Write(buffer)
		require.NoError(t, err, "test peer could not write to local transport")
		require.Equal(t, len(buffer), written)

		buffer = []byte{0}
		read, err := h.peerTalkerConnection.Read(buffer)
		require.Equal(t, 0, read, "test peer should be disconnected from local transport without reading anything")
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
			require.NoError(t, err, "test peer server could not read from local transport")
			require.Equal(t, []byte{0x00, 0x00, 0x00, 0x00}, data) // keepalive content (zero payloads)
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
			buffer := []byte{0x00, 0x00, 0x00, 0x00} // keepalive content (zero payloads)
			written, err := h.peerTalkerConnection.Write(buffer)
			require.NoError(t, err, "test peer could not write to local transport")
			require.Equal(t, len(buffer), written)
		}

		buffer := []byte{0x02, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x11, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x22, 0x33, 0x00, 0x00}
		written, err := h.peerTalkerConnection.Write(buffer)
		require.NoError(t, err, "test peer could not write to local transport")
		require.Equal(t, len(buffer), written)

		h.verifyTransportListenerCalled(t)
	})
}
