// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const someAddress = ""

func TestQueue_PushAndPopMultiple(t *testing.T) {
	with.Context(func(ctx context.Context) {
		q := aQueue(t, 1000, 1000)

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
	})
}

func TestQueue_CannotPushMoreThanMaxMessages(t *testing.T) {
	with.Context(func(ctx context.Context) {
		q := aQueue(t, 1000, 2)

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
	})
}

func TestQueue_PopWhenEmptyWaitsUntilPush(t *testing.T) {
	with.Context(func(ctx context.Context) {
		q := aQueue(t, 1000, 1000)

		go func() {
			time.Sleep(10 * time.Millisecond)
			err := q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x01}})
			require.NoError(t, err)
		}()

		d1 := q.Pop(ctx)
		require.EqualValues(t, []byte{0x01}, d1.SenderNodeAddress)
	})
}

func TestQueue_PopWhenEmptyCancelsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	q := aQueue(t, 1000, 1000)

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	d1 := q.Pop(ctx)
	require.Nil(t, d1)
}

func TestQueue_CannotPushMoreThanMaxBytes(t *testing.T) {
	with.Context(func(ctx context.Context) {
		q := aQueue(t, 10, 1000)

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
	})
}

func TestQueue_ClearEmptiesTheQueue(t *testing.T) {
	with.Context(func(ctx context.Context) {
		q := aQueue(t, 1000, 3)

		q.Clear(ctx)

		err := q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x01}})
		require.NoError(t, err)

		err = q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x02}})
		require.NoError(t, err)

		err = q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x03}})
		require.NoError(t, err)

		q.Clear(ctx)

		err = q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x01}})
		require.NoError(t, err)

		err = q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x02}})
		require.NoError(t, err)

		err = q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x03}})
		require.NoError(t, err)
	})
}

func TestQueue_DisableThenEnable(t *testing.T) {
	with.Context(func(ctx context.Context) {
		q := aQueue(t, 1000, 2)

		q.Disable()

		err := q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x01}})
		require.Error(t, err)

		q.Enable()

		err = q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x02}})
		require.NoError(t, err)

		err = q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x03}})
		require.NoError(t, err)

		q.Disable()

		err = q.Push(&adapter.TransportData{SenderNodeAddress: []byte{0x04}})
		require.Error(t, err)
	})
}

func buf(len int) []byte {
	return make([]byte, len)
}

func aQueue(t testing.TB, maxSizeInBytes int, maxNumOfMessages int) *transportQueue {
	return NewTransportQueue(maxSizeInBytes, maxNumOfMessages, metric.NewRegistry(), someAddress)
}
