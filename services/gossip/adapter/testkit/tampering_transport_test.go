package testkit

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/memory"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"sync"
	"testing"
)

type tamperingHarness struct {
	senderKey string
	transport *TamperingTransport
	listener  *MockTransportListener
}

func newTamperingHarness(tb testing.TB, ctx context.Context) *tamperingHarness {
	senderAddress := "sender"
	listenerAddress := "listener"
	listener := &MockTransportListener{}
	logger := log.DefaultTestingLogger(tb).WithTags(log.String("adapter", "transport"))

	federationNodes := make(map[string]config.FederationNode)
	federationNodes[senderAddress] = config.NewHardCodedFederationNode(primitives.NodeAddress(senderAddress))
	federationNodes[listenerAddress] = config.NewHardCodedFederationNode(primitives.NodeAddress(listenerAddress))

	transport := NewTamperingTransport(logger, memory.NewTransport(ctx, logger, federationNodes))

	transport.RegisterListener(listener, primitives.NodeAddress(listenerAddress))

	return &tamperingHarness{
		senderKey: senderAddress,
		transport: transport,
		listener:  listener,
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

func TestFailingTamperer(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		c := newTamperingHarness(t, ctx)

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
	test.WithContext(func(ctx context.Context) {
		c := newTamperingHarness(t, ctx)

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

func TestLatchingTamperer(t *testing.T) {
	t.Skip("this test is suspect as having a deadlock, skipping until @ronnno and @electricmonk can look at it")
	test.WithContext(func(ctx context.Context) {
		c := newTamperingHarness(t, ctx)

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
