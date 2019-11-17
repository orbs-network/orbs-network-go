// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package testkit

import (
	"context"
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/memory"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

type tamperingHarness struct {
	*with.ConcurrencyHarness
	senderKey string
	transport *TamperingTransport
	listeners []*MockTransportListener
	t         *testing.T

	mutex sync.Mutex
}

func newTamperingHarness(t *testing.T, parent *with.ConcurrencyHarness, ctx context.Context, numListeners int) *tamperingHarness {
	senderAddress := "sender"
	genesisValidatorNodes := make(map[string]config.ValidatorNode)
	genesisValidatorNodes[senderAddress] = config.NewHardCodedValidatorNode(primitives.NodeAddress(senderAddress))
	listeners := []*MockTransportListener{}
	addresses := []string{}

	for i := 0; i < numListeners; i++ {
		listenerAddress := fmt.Sprintf("listener%d", i)
		addresses = append(addresses, listenerAddress)
		listener := &MockTransportListener{}
		listeners = append(listeners, listener)
		genesisValidatorNodes[listenerAddress] = config.NewHardCodedValidatorNode(primitives.NodeAddress(listenerAddress))
	}

	memoryTransport := memory.NewTransport(ctx, parent.Logger, genesisValidatorNodes)
	transport := NewTamperingTransport(parent.Logger, memoryTransport)
	for ind, listener := range listeners {
		transport.RegisterListener(listener, primitives.NodeAddress(addresses[ind]))
	}

	harness := &tamperingHarness{
		ConcurrencyHarness: parent,
		senderKey:          senderAddress,
		transport:          transport,
		listeners:          listeners,
		t:                  t,
		mutex:              sync.Mutex{},
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

func (c *tamperingHarness) verify() {
	for _, listener := range c.listeners {
		ok, err := listener.Verify()
		if !ok {
			c.t.Errorf(err.Error())
		}
	}
}

func withTamperingHarness(ctx context.Context, t *testing.T, parent *with.ConcurrencyHarness, numNodes int, f func(*tamperingHarness)) {
	c := newTamperingHarness(t, parent, ctx, numNodes)
	f(c)
	c.verify()
}

func TestFailingTamperer(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		withTamperingHarness(ctx, t, parent, 1, func(c *tamperingHarness) {
			c.transport.Fail(anyMessage())

			c.listeners[0].ExpectNotReceive()
			c.send(ctx, nil)

			time.Sleep(50 * time.Millisecond) // TODO we want to make sure something never happens - how long do we wait? instead, wait until the transport is done emitting transmissions
		})
	})
}

func TestFailingTamperer_DoesPartialBroadcast(t *testing.T) {
	const numNodes = 2
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		withTamperingHarness(ctx, t, parent, numNodes, func(c *tamperingHarness) {
			const iterations = 5

			signals := make(chan byte, iterations*numNodes)

			c.transport.Fail(everySecondTime())
			c.listeners[0].WhenOnTransportMessageReceived(mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
				signals <- payloads[0][0]
			}).AtMost(iterations)
			c.listeners[1].WhenOnTransportMessageReceived(mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
				signals <- payloads[0][0]
			}).AtMost(iterations)

			for i := byte(0); i < iterations; i++ {
				c.send(ctx, [][]byte{{i}})
				n := <-signals
				require.Equal(t, i, n, "expected the current iteration number")
			}

			timeout, cancel := context.WithTimeout(ctx, 50*time.Millisecond) // TODO we want to make sure something never happens - how long do we wait? instead, wait until the transport is done emitting transmissions
			defer cancel()
			select {
			case <-timeout.Done(): // OK
			case <-signals:
				t.Fatalf("expected no more signals to arrive after reading %d signals", iterations)
			}
		})
	})
}

// Make sure the payload is cloned and corrupted per transmission
func TestCorruptingTamperer_DoesPartialBroadcast(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		withTamperingHarness(ctx, t, parent, 2, func(c *tamperingHarness) {
			const iterations = 5
			signals := make(chan byte, 2)

			ctrlRand := rand.NewControlledRand(t)
			c.transport.Corrupt(everySecondTimeFlipped(), ctrlRand)

			c.listeners[0].WhenOnTransportMessageReceived(mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
				signals <- payloads[0][0]
			}).Times(iterations)
			c.listeners[1].WhenOnTransportMessageReceived(mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
				signals <- payloads[0][0]
			}).Times(iterations)

			for i := 0; i < iterations; i++ {
				c.send(ctx, [][]byte{{0}})
				n1 := <-signals
				n2 := <-signals
				require.True(t, n1 == 0 && n2 != 0 || n1 != 0 && n2 == 0,
					fmt.Sprintf("expected a corruption of exactly one message, got 0x%x 0x%x", n1, n2))
			}
		})
	})
}

func TestCorruptingTamperer_PreserveInputPayload(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		withTamperingHarness(ctx, t, parent, 1, func(c *tamperingHarness) {
			const iterations = 3

			signals := make(chan byte, 1)

			ctrlRand := rand.NewControlledRand(t)
			c.transport.Corrupt(anyMessage(), ctrlRand)

			c.listeners[0].WhenOnTransportMessageReceived(mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
				signals <- payloads[0][0]
			}).Times(iterations)

			for i := 0; i < iterations; i++ {
				payloads := [][]byte{{0}}
				c.send(ctx, payloads)
				n := <-signals
				require.NotEqual(t, byte(0), n, "expected received payload to be corrupted")
				require.Equal(t, byte(0), payloads[0][0], "expected input payload to preserve its original value")
			}
		})
	})
}

func TestPausingTamperer(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		withTamperingHarness(ctx, t, parent, 1, func(c *tamperingHarness) {
			const sendCount = 10 // Must be even
			digits := make(chan byte, sendCount)

			odds := c.transport.Pause(oddNumbers())

			c.listeners[0].WhenOnTransportMessageReceived(mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
				digits <- payloads[0][0]
			}).Times(sendCount)

			for b := 0; b < sendCount; b++ {
				c.send(ctx, [][]byte{{byte(b)}})
			}

			for b := 0; b < sendCount/2; b++ {
				if <-digits%2 != 0 {
					t.Errorf("got odd number while odds should be paused")
				}
			}

			odds.StopTampering(ctx)

			for b := 0; b < sendCount/2; b++ {
				if <-digits%2 != 1 {
					t.Errorf("got even number while odds should be released")
				}
			}
		})
	})
}

// this test is suspect as having a deadlock, may need to skip it
func TestLatchingTamperer(t *testing.T) {
	t.Skip("this test is suspect as having a deadlock, skipping until @ronnno and @electricmonk can look at it; handled in https://github.com/orbs-network/orbs-network-go/pull/769")
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		withTamperingHarness(ctx, t, parent, 1, func(c *tamperingHarness) {
			called := make(chan bool)

			latch := c.transport.LatchOn(anyMessage())

			c.listeners[0].WhenOnTransportMessageReceived(mock.Any)

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

func everySecondTime() MessagePredicate {
	count := 0
	return func(data *adapter.TransportData) bool {
		count += 1
		return count%2 == 0
	}
}

// true, false, false, true, true, false, ....
func everySecondTimeFlipped() MessagePredicate {
	count := -1
	return func(data *adapter.TransportData) bool {
		count += 1
		return count%2 == count/2%2
	}
}
