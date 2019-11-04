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
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

type tamperingHarness struct {
	senderKey      string
	transport      *TamperingTransport
	firstListener  *MockTransportListener
	secondListener *MockTransportListener
	t              *testing.T
}

func newTamperingHarness(t *testing.T, logger log.Logger, ctx context.Context) *tamperingHarness {
	senderAddress := "sender"
	firstListenerAddress := "listener1"
	secondListenerAddress := "listener2"
	firstListener := &MockTransportListener{}
	secondListener := &MockTransportListener{}

	genesisValidatorNodes := make(map[string]config.ValidatorNode)
	genesisValidatorNodes[senderAddress] = config.NewHardCodedValidatorNode(primitives.NodeAddress(senderAddress))
	genesisValidatorNodes[firstListenerAddress] = config.NewHardCodedValidatorNode(primitives.NodeAddress(firstListenerAddress))
	genesisValidatorNodes[secondListenerAddress] = config.NewHardCodedValidatorNode(primitives.NodeAddress(secondListenerAddress))

	transport := NewTamperingTransport(logger, memory.NewTransport(ctx, logger, genesisValidatorNodes))

	transport.RegisterListener(firstListener, primitives.NodeAddress(firstListenerAddress))
	transport.RegisterListener(secondListener, primitives.NodeAddress(secondListenerAddress))

	return &tamperingHarness{
		senderKey:      senderAddress,
		transport:      transport,
		firstListener:  firstListener,
		secondListener: secondListener,
		t:              t,
	}
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
	for _, listener := range []*MockTransportListener{c.firstListener, c.secondListener} {
		ok, err := listener.Verify()
		if !ok {
			c.t.Errorf(err.Error())
		}
	}
}

func withTamperingHarness(ctx context.Context, t *testing.T, logger log.Logger, f func(*tamperingHarness)) {
	c := newTamperingHarness(t, logger, ctx)
	f(c)
	c.verify()
}

func TestFailingTamperer(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			withTamperingHarness(ctx, t, parent.Logger, func(c *tamperingHarness) {
				c.transport.Fail(anyMessage())

				c.send(ctx, nil)

				c.firstListener.ExpectNotReceive()

				ok, err := c.firstListener.Verify()
				if !ok {
					t.Fatal(err)
				}
			})
		})
	})
}

func TestFailingTamperer_DoesPartialBroadcast(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			withTamperingHarness(ctx, t, parent.Logger, func(c *tamperingHarness) {
				const iterations = 5

				const numNodes = 2
				signals := make(chan byte, iterations*numNodes)

				c.transport.Fail(everySecondTime())
				c.firstListener.WhenOnTransportMessageReceived(mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
					signals <- payloads[0][0]
				}).AtMost(iterations)
				c.secondListener.WhenOnTransportMessageReceived(mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
					signals <- payloads[0][0]
				}).AtMost(iterations)

				for i := byte(0); i < iterations; i++ {
					c.send(ctx, [][]byte{{i}})
					n := <-signals
					require.Equal(t, i, n, "expected the current iteration number")
				}

				timeout, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
				defer cancel()
				select {
				case <-timeout.Done(): // OK
				case <-signals:
					t.Fatalf("expected no more signals to arrive after reading %d signals", iterations)
				}
			})
		})
	})
}

func TestCorruptingTamperer_DoesPartialBroadcast(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			withTamperingHarness(ctx, t, parent.Logger, func(c *tamperingHarness) {
				const iterations = 5
				signals := make(chan byte, 2)

				ctrlRand := rand.NewControlledRand(t)
				c.transport.Corrupt(everySecondTimeFlipped(), ctrlRand)

				c.firstListener.WhenOnTransportMessageReceived(mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
					signals <- payloads[0][0]
				}).Times(iterations)
				c.secondListener.WhenOnTransportMessageReceived(mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
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
	})
}

func TestCorruptingTamperer_PreserveInputPayload(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			withTamperingHarness(ctx, t, parent.Logger, func(c *tamperingHarness) {
				const iterations = 3

				signals := make(chan byte, 2)

				ctrlRand := rand.NewControlledRand(t)
				c.transport.Corrupt(anyMessage(), ctrlRand)

				c.firstListener.WhenOnTransportMessageReceived(mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
					signals <- payloads[0][0]
				}).Times(iterations)
				c.secondListener.WhenOnTransportMessageReceived(mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
					signals <- payloads[0][0]
				}).Times(iterations)

				for i := 0; i < iterations; i++ {
					payloads := [][]byte{{0}}
					c.send(ctx, payloads)
					for l := 0; l < 2; l++ {
						n := <-signals
						require.NotEqual(t, byte(0), n, "expected received payload to be corrupted")
					}
					require.Equal(t, byte(0), payloads[0][0], "expected input payload to preserve its original value")
				}
			})
		})
	})
}

func TestPausingTamperer(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			withTamperingHarness(ctx, t, parent.Logger, func(c *tamperingHarness) {
				digits := make(chan byte, 10)

				odds := c.transport.Pause(oddNumbers())

				c.firstListener.WhenOnTransportMessageReceived(mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
					digits <- payloads[0][0]
				}).Times(10)

				c.secondListener.WhenOnTransportMessageReceived(mock.Any).Call(func(ctx context.Context, payloads [][]byte) {
					digits <- payloads[0][0]
				}).Times(10)

				for b := 0; b < 10; b++ {
					c.send(ctx, [][]byte{{byte(b)}})
				}

				for b := 0; b < 10; b++ {
					if <-digits%2 != 0 {
						t.Errorf("got odd number while odds should be paused")
					}
				}

				odds.StopTampering(ctx)

				for b := 0; b < 10; b++ {
					if <-digits%2 != 1 {
						t.Errorf("got even number while odds should be released")
					}
				}
			})
		})
	})
}

// this test is suspect as having a deadlock, may need to skip it
func TestLatchingTamperer(t *testing.T) {
	t.Skip("this test is suspect as having a deadlock, skipping until @ronnno and @electricmonk can look at it; handled in https://github.com/orbs-network/orbs-network-go/pull/769")
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			withTamperingHarness(ctx, t, parent.Logger, func(c *tamperingHarness) {
				called := make(chan bool)

				latch := c.transport.LatchOn(anyMessage())

				c.firstListener.WhenOnTransportMessageReceived(mock.Any)

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
