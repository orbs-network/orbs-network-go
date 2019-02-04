package tcp

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestQueue_PushAndPopMultiple(t *testing.T) {
	ctx := context.Background()
	q := NewTransportQueue(1000, 1000)

	err := q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x01}})
	require.NoError(t, err)

	err = q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x02}})
	require.NoError(t, err)

	d1 := q.Pop(ctx)
	require.EqualValues(t, []byte{0x01}, d1.SenderNodeAddress)

	err = q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x03}})
	require.NoError(t, err)

	d2 := q.Pop(ctx)
	require.EqualValues(t, []byte{0x02}, d2.SenderNodeAddress)

	d3 := q.Pop(ctx)
	require.EqualValues(t, []byte{0x03}, d3.SenderNodeAddress)
}

func TestQueue_CannotPushMoreThanMaxMessages(t *testing.T) {
	ctx := context.Background()
	q := NewTransportQueue(1000, 2)

	err := q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x01}})
	require.NoError(t, err)

	err = q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x02}})
	require.NoError(t, err)

	err = q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x03}})
	require.Error(t, err, "queue should be full")

	err = q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x04}})
	require.Error(t, err, "queue should be full")

	d1 := q.Pop(ctx)
	require.EqualValues(t, []byte{0x01}, d1.SenderNodeAddress)

	err = q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x03}})
	require.NoError(t, err)
}

func TestQueue_PopWhenEmptyWaitsUntilPush(t *testing.T) {
	ctx := context.Background()
	q := NewTransportQueue(1000, 1000)

	go func() {
		time.Sleep(10 * time.Millisecond)
		err := q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x01}})
		require.NoError(t, err)
	}()

	d1 := q.Pop(ctx)
	require.EqualValues(t, []byte{0x01}, d1.SenderNodeAddress)
}

func TestQueue_PopWhenEmptyCancelsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	q := NewTransportQueue(1000, 1000)

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	d1 := q.Pop(ctx)
	require.Nil(t, d1)
}

func TestQueue_CannotPushMoreThanMaxBytes(t *testing.T) {
	ctx := context.Background()
	q := NewTransportQueue(10, 1000)

	err := q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x01}, Payloads: [][]byte{buf(3), buf(4)}})
	require.NoError(t, err)

	err = q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x02}, Payloads: [][]byte{buf(1), buf(6)}})
	require.Error(t, err, "queue should be full")

	err = q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x03}, Payloads: [][]byte{buf(4)}})
	require.Error(t, err, "queue should be full")

	err = q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x04}, Payloads: [][]byte{buf(3)}})
	require.NoError(t, err)

	d1 := q.Pop(ctx)
	require.EqualValues(t, []byte{0x01}, d1.SenderNodeAddress)

	err = q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x05}, Payloads: [][]byte{buf(1), buf(6)}})
	require.NoError(t, err)
}

func buf(len int) []byte {
	return make([]byte, len)
}
