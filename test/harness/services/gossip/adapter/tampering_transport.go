package adapter

import (
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

// The TamperingTransport is an in-memory implementation of the Gossip Transport adapter, that adds the ability
// to tamper with the messages or to synchronize the test's goroutine with the SUT's goroutines
type TamperingTransport interface {
	adapter.Transport

	// Creates an ongoing tamper which fails messages matching the given predicate, returning an error object to the sender.
	// This is useful to emulate network errors, for instance
	Fail(predicate MessagePredicate) OngoingTamper

	// Creates an ongoing tamper which delays messages matching the given predicate. The messages will be sent when
	// calling OngoingTamper.Release(). This is useful for emulating network congestion or messages arriving in an order
	// different than expected
	Pause(predicate MessagePredicate) OngoingTamper

	// Creates an ongoing tamper which latches the latching goroutine (typically a test) until at least one message
	// matching the given predicate is sent. The latch is created as inactive, and will only block the caller after
	// calling LatchingTamper.Wait(). This is useful to force a test goroutine to block until a certain message has
	// been sent
	LatchOn(predicate MessagePredicate) LatchingTamper

	// Creates an ongoing tamper which duplicates messages matching the given predicate
	Duplicate(predicate MessagePredicate) OngoingTamper

	// Creates an ongoing tamper which corrupts messages matching the given predicate
	Corrupt(predicate MessagePredicate) OngoingTamper

	// Creates an ongoing tamper which delays (reshuffles) messages matching the given predicate for a random duration
	Delay(predicate MessagePredicate) OngoingTamper
}

// A predicate for matching messages with a certain property
type MessagePredicate func(data *adapter.TransportData) bool

type OngoingTamper interface {
	Release()
	maybeTamper(data *adapter.TransportData) (error, bool)
}

type LatchingTamper interface {
	Wait()
	Remove()
}

type tamperingTransport struct {
	mutex                *sync.Mutex
	transportListeners   map[string]adapter.TransportListener
	latchingTamperers    []*latchingTamperer

	ongoingTamperers	 []OngoingTamper
}

func NewTamperingTransport() TamperingTransport {
	return &tamperingTransport{
		transportListeners: make(map[string]adapter.TransportListener),
		mutex:              &sync.Mutex{},
	}
}

func (t *tamperingTransport) RegisterListener(listener adapter.TransportListener, listenerPublicKey primitives.Ed25519PublicKey) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.transportListeners[string(listenerPublicKey)] = listener
}

func (t *tamperingTransport) Send(data *adapter.TransportData) error {
	t.releaseLatches(data)

	if err, shouldReturn := t.maybeTamper(data); shouldReturn {
		return err
	}

	go t.receive(data)
	return nil
}

func (t *tamperingTransport) maybeTamper(data *adapter.TransportData) (error, bool) {
	for _, o := range t.ongoingTamperers {
		if err, shouldReturn := o.maybeTamper(data); shouldReturn {
			return err, shouldReturn
		}
	}

	return nil, false

}

func (t *tamperingTransport) Pause(predicate MessagePredicate) OngoingTamper {
	return t.addTamperer(&pausingTamperer{predicate: predicate, transport: t})
}

func (t *tamperingTransport) Fail(predicate MessagePredicate) OngoingTamper {
	return t.addTamperer(&failingTamperer{predicate: predicate, transport: t})
}

func (t *tamperingTransport) Duplicate(predicate MessagePredicate) OngoingTamper {
	return t.addTamperer(&duplicatingTamperer{predicate: predicate, transport: t})
}

func (t *tamperingTransport) Corrupt(predicate MessagePredicate) OngoingTamper {
	return t.addTamperer(&corruptingTamperer{predicate: predicate, transport: t})
}

func (t *tamperingTransport) Delay(predicate MessagePredicate) OngoingTamper {
	return t.addTamperer(&delayingTamperer{predicate: predicate, transport: t})
}

func (t *tamperingTransport) LatchOn(predicate MessagePredicate) LatchingTamper {
	tamperer := &latchingTamperer{predicate: predicate, transport: t, cond: sync.NewCond(&sync.Mutex{})}
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.latchingTamperers = append(t.latchingTamperers, tamperer)

	tamperer.cond.L.Lock()
	return tamperer
}

func (t *tamperingTransport) removeOngoingTamperer(tamperer OngoingTamper) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	a := t.ongoingTamperers
	for p, v := range a {
		if v == tamperer {
			a[p] = a[len(a)-1]
			a[len(a)-1] = nil
			a = a[:len(a)-1]

			t.ongoingTamperers = a

			return
		}
	}
	panic("Tamperer not found in ongoing tamperer list")
}

func (t *tamperingTransport) removeLatchingTamperer(tamperer *latchingTamperer) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	a := t.latchingTamperers
	for p, v := range a {
		if v == tamperer {
			a[p] = a[len(a)-1]
			a[len(a)-1] = nil
			a = a[:len(a)-1]

			t.latchingTamperers = a

			return
		}
	}
	panic("Tamperer not found in ongoing tamperer list")
}

func (t *tamperingTransport) receive(data *adapter.TransportData) {
	switch data.RecipientMode {

	case gossipmessages.RECIPIENT_LIST_MODE_BROADCAST:
		t.mutex.Lock()
		defer t.mutex.Unlock()

		for stringPublicKey, l := range t.transportListeners {
			if stringPublicKey != string(data.SenderPublicKey) {
				l.OnTransportMessageReceived(data.Payloads)
			}
		}

	case gossipmessages.RECIPIENT_LIST_MODE_LIST:
		t.mutex.Lock()
		defer t.mutex.Unlock()

		for _, recipientPublicKey := range data.RecipientPublicKeys {
			stringPublicKey := string(recipientPublicKey)
			t.transportListeners[stringPublicKey].OnTransportMessageReceived(data.Payloads)
		}

	case gossipmessages.RECIPIENT_LIST_MODE_ALL_BUT_LIST:
		panic("Not implemented")
	}

}

func (t *tamperingTransport) releaseLatches(data *adapter.TransportData) {
	for _, o := range t.latchingTamperers {
		if o.predicate(data) {
			o.cond.Signal()
		}
	}
}

func (t *tamperingTransport) addTamperer(tamperer OngoingTamper) OngoingTamper {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.ongoingTamperers = append(t.ongoingTamperers, tamperer)
	return tamperer
}

type failingTamperer struct {
	predicate MessagePredicate
	transport *tamperingTransport
}

func (o *failingTamperer) maybeTamper(data *adapter.TransportData) (error, bool) {
	if o.predicate(data) {
		return &adapter.ErrTransportFailed{data}, true
	}

	return nil, false
}

func (o *failingTamperer) Release() {
	o.transport.removeOngoingTamperer(o)
}

type duplicatingTamperer struct {
	predicate MessagePredicate
	transport *tamperingTransport
}

func (o *duplicatingTamperer) maybeTamper(data *adapter.TransportData) (error, bool) {
	if o.predicate(data) {
		go func() {
			time.Sleep(10 * time.Millisecond)
			o.transport.receive(data)
		}()
	}
	return nil, false
}

func (o *duplicatingTamperer) Release() {
	o.transport.removeOngoingTamperer(o)
}

type delayingTamperer struct {
	predicate MessagePredicate
	transport *tamperingTransport
}

func (o *delayingTamperer) maybeTamper(data *adapter.TransportData) (error, bool) {
	if o.predicate(data) {
		duration := time.Duration(rand.Intn(10000)) * time.Microsecond
		go func() {
			time.Sleep(duration)
			o.transport.receive(data)
		}()
		return nil, true
	}

	return nil, false
}

func (o *delayingTamperer) Release() {
	o.transport.removeOngoingTamperer(o)
}

type corruptingTamperer struct {
	predicate MessagePredicate
	transport *tamperingTransport
}

func (o *corruptingTamperer) maybeTamper(data *adapter.TransportData) (error, bool) {
	if o.predicate(data) {
		for i := 0; i < 10; i++ {
			x := rand.Intn(len(data.Payloads))
			y := rand.Intn(len(data.Payloads[x]))
			data.Payloads[x][y] = 0
		}
	}
	return nil, false
}

func (o *corruptingTamperer) Release() {
	o.transport.removeOngoingTamperer(o)
}

type pausingTamperer struct {
	predicate MessagePredicate
	transport *tamperingTransport
	messages  []*adapter.TransportData
}

func (o *pausingTamperer) maybeTamper(data *adapter.TransportData) (error, bool) {
	if o.predicate(data) {
		o.transport.mutex.Lock()
		defer o.transport.mutex.Unlock()
		o.messages = append(o.messages, data)
		return nil, true
	}

	return nil, false
}

func (o *pausingTamperer) Release() {
	o.transport.removeOngoingTamperer(o)
	for _, message := range o.messages {
		o.transport.Send(message)
		runtime.Gosched() // TODO: this is required or else messages arrive in the opposite order after resume
	}
}

type latchingTamperer struct {
	predicate MessagePredicate
	transport *tamperingTransport
	cond      *sync.Cond
}

func (o *latchingTamperer) Remove() {
	o.transport.removeLatchingTamperer(o)
}

func (o *latchingTamperer) Wait() {
	o.cond.Wait() // TODO: change cond to channel close
}
