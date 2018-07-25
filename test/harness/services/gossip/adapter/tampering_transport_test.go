package adapter

import (
	"testing"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/go-mock"
		"sync"
	)

type context struct {
	senderKey string
	transport TamperingTransport
	listener  *mockListener
}

func newContext() *context {
	senderKey := "sender"
	listenerKey := "listener"
	listener := &mockListener{}
	transport := NewTamperingTransport()
	transport.RegisterListener(listener, primitives.Ed25519Pkey(listenerKey))

	return &context{
		senderKey: senderKey,
		transport: transport,
		listener: listener,
	}
}

func (c *context) send(payloads [][]byte) {
	c.broadcast(c.senderKey, payloads)
}

func (c *context) broadcast(sender string, payloads [][]byte) error {
	return c.transport.Send(&adapter.TransportData{
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		SenderPublicKey: primitives.Ed25519Pkey(sender),
		Payloads:        payloads,
	})
}


func TestFailingTamperer(t *testing.T) {
	c := newContext()

	c.transport.Fail(anyMessage())

	c.send(nil)

	c.listener.expectNotReceive()

	ok, err := c.listener.Verify()
	if !ok {
		t.Fatal(err)
	}
}

func TestPausingTamperer(t *testing.T) {
	c := newContext()

	digits := make(chan byte, 10)

	odds := c.transport.Pause(oddNumbers())

	c.listener.WhenOnTransportMessageReceived(mock.Any).Call(func(payloads [][]byte) {
		digits <- payloads[0][0]
	}).Times(10)

	for b := 0; b < 10; b++ {
		c.send([][]byte{{byte(b)}})
	}

	for b := 0; b < 5; b++ {
		if <-digits % 2 != 0 {
			t.Errorf("got odd number while odds should be paused")
		}
	}

	odds.Release()

	for b := 0; b < 5; b++ {
		if <-digits % 2 != 1 {
			t.Errorf("got even number while odds should be released")
		}
	}
}

func TestLatchingTamperer(t *testing.T) {
	c := newContext()

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
	c.send(nil)

	select {
	case <- called:
		t.Error("called too early")
	default:
	}

	afterMessageArrived.Done()

	<- called
}

func oddNumbers() MessagePredicate {
	return func(data *adapter.TransportData) bool {
		return data.Payloads[0][0] % 2 == 1
	}
}

func anyMessage() MessagePredicate {
	return func(data *adapter.TransportData) bool {
		return true
	}
}

