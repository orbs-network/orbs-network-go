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
	require.NoError(t, err, "test should be able connect to local transport")

	cancel()

	buffer := []byte{0}
	read, err := connection.Read(buffer)
	require.Equal(t, 0, read, "test should disconnect from local transport without reading anything")
	require.Error(t, err, "test should disconnect from local transport")

	_, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", h.myPort))
	require.Error(t, err, "test should be able to connect to local transport")
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

		for i := 0; i < 2; i++ {
			connection.Close() // disconnect local transport forcefully

			connection, err = listener.Accept()
			require.NoError(t, err, "test peer server could not accept connection from local transport")
		}
	})
}

func TestOutgoing_AdapterBroadcast(t *testing.T) {
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
			require.NoError(t, err, "test peer server could not accept connection from local transport")
			require.Equal(t, []byte{0x02, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x11, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x22, 0x33, 0x00, 0x00}, data)
		}
	})
}

func TestOutgoing_AdapterUnicast(t *testing.T) {
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
		require.NoError(t, err, "test peer server could not accept connection from local transport")
		require.Equal(t, []byte{0x02, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x11, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x22, 0x33, 0x00, 0x00}, data)
	})
}
