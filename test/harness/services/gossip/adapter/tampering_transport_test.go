package adapter

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"sync"
	"testing"
)

type tamperingHarness struct {
	senderKey string
	transport TamperingTransport
	listener  *mockListener
}

func newTamperingHarness() *tamperingHarness {
	senderKey := "sender"
	listenerKey := "listener"
	listener := &mockListener{}
	transport := NewTamperingTransport(log.GetLogger(log.String("adapter", "transport")))
	transport.RegisterListener(listener, primitives.Ed25519PublicKey(listenerKey))

	return &tamperingHarness{
		senderKey: senderKey,
		transport: transport,
		listener:  listener,
	}
}

func (c *tamperingHarness) send(ctx context.Context, payloads [][]byte) {
	c.broadcast(ctx, c.senderKey, payloads)
}

func (c *tamperingHarness) broadcast(ctx context.Context, sender string, payloads [][]byte) error {
	return c.transport.Send(ctx, &adapter.TransportData{
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		SenderPublicKey: primitives.Ed25519PublicKey(sender),
		Payloads:        payloads,
	})
}

func TestFailingTamperer(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		c := newTamperingHarness()

		c.transport.Fail(anyMessage())

		c.send(ctx, nil)

		c.listener.expectNotReceive()

		ok, err := c.listener.Verify()
		if !ok {
			t.Fatal(err)
		}
	})
}

func TestPausingTamperer(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		c := newTamperingHarness()

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

		odds.Release(ctx)

		for b := 0; b < 5; b++ {
			if <-digits%2 != 1 {
				t.Errorf("got even number while odds should be released")
			}
		}
	})
}

func TestLatchingTamperer(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		c := newTamperingHarness()

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
