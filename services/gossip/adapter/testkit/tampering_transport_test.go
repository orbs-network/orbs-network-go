// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package testkit

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/memory"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"sync"
	"testing"
)

type tamperingHarness struct {
	*with.ConcurrencyHarness
	senderKey string
	transport *TamperingTransport
	listener  *MockTransportListener
}

func newTamperingHarness(ctx context.Context, parent *with.ConcurrencyHarness) *tamperingHarness {
	senderAddress := "sender"
	listenerAddress := "listener"
	listener := &MockTransportListener{}

	genesisValidatorNodes := make(map[string]config.ValidatorNode)
	genesisValidatorNodes[senderAddress] = config.NewHardCodedValidatorNode(primitives.NodeAddress(senderAddress))
	genesisValidatorNodes[listenerAddress] = config.NewHardCodedValidatorNode(primitives.NodeAddress(listenerAddress))

	memoryTransport := memory.NewTransport(ctx, parent.Logger, genesisValidatorNodes)
	transport := NewTamperingTransport(parent.Logger, memoryTransport)

	transport.RegisterListener(listener, primitives.NodeAddress(listenerAddress))

	harness := &tamperingHarness{
		ConcurrencyHarness: parent,
		senderKey:          senderAddress,
		transport:          transport,
		listener:           listener,
	}

	harness.Supervise(memoryTransport)

	return harness
}

func (c *tamperingHarness) send(ctx context.Context, payloads [][]byte) {
	c.broadcast(ctx, c.senderKey, payloads)
}

func (c *tamperingHarness) broadcast(ctx context.Context, sender string, payloads [][]byte) error {
	return c.transport.Send(ctx, &adapter.TransportData{
		RecipientMode:     gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		SenderNodeAddress: primitives.NodeAddress(sender),
		Payloads:          payloads,
	})
}

func TestFailingTamperer(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		c := newTamperingHarness(ctx, parent)

		c.transport.Fail(anyMessage())

		c.send(ctx, nil)

		c.listener.ExpectNotReceive()

		ok, err := c.listener.Verify()
		if !ok {
			t.Fatal(err)
		}
	})
}

func TestPausingTamperer(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		c := newTamperingHarness(ctx, parent)

		digits := make(chan byte, 10)
		odds := c.transport.Pause(oddNumbers())

		c.listener.WhenOnTransportMessageReceived(mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
			digits <- payloads[0][0]
		}).Times(10)

		for b := 0; b < 10; b++ {
			c.send(ctx, [][]byte{{byte(b)}})
		}

		for b := 0; b < 5; b++ {
			if <-digits%2 != 0 {
				t.Errorf("got odd number while odds should be paused")
			}
		}

		odds.StopTampering(ctx)

		for b := 0; b < 5; b++ {
			if <-digits%2 != 1 {
				t.Errorf("got even number while odds should be released")
			}
		}
	})
}

// this test is suspect as having a deadlock, may need to skip it
func TestLatchingTamperer(t *testing.T) {
	t.Skip("this test is suspect as having a deadlock, skipping until @ronnno and @electricmonk can look at it; handled in https://github.com/orbs-network/orbs-network-go/pull/769")
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {

		c := newTamperingHarness(ctx, parent)

		called := make(chan bool)

		latch := c.transport.LatchOn(anyMessage())

		c.listener.WhenOnTransportMessageReceived(mock.Any)

		afterMessageArrived := sync.WaitGroup{}
		afterMessageArrived.Add(1)

		afterLatched := sync.WaitGroup{}
		afterLatched.Add(1)

		go func() {
			afterLatched.Done()

			defer func() {
				latch.Wait()
				afterMessageArrived.Wait()
				called <- true
			}()

		}()

		afterLatched.Wait()
		c.send(ctx, nil)

		select {
		case <-called:
			t.Error("called too early")
		default:
		}

		afterMessageArrived.Done()

		<-called
	})
}

func oddNumbers() MessagePredicate {
	return func(data *adapter.TransportData) bool {
		return data.Payloads[0][0]%2 == 1
	}
}

func anyMessage() MessagePredicate {
	return func(data *adapter.TransportData) bool {
		return true
	}
}
