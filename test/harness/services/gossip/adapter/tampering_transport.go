package adapter

import (
	"context"
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

	// Creates an ongoing tamper which delays (reshuffles) messages matching the given predicate for the specified duration
	Delay(duration func() time.Duration, predicate MessagePredicate) OngoingTamper
}

// A predicate for matching messages with a certain property
type MessagePredicate func(data *adapter.TransportData) bool

type OngoingTamper interface {
	Release(ctx context.Context)
	maybeTamper(ctx context.Context, data *adapter.TransportData) (error, bool)
}

type LatchingTamper interface {
	Wait()
	Remove()
}

type tamperingTransport struct {
	listenerLock                 *sync.RWMutex
	transportListenersUnderMutex map[string]adapter.TransportListener

	tampererLock                *sync.RWMutex
	latchingTamperersUnderMutex []*latchingTamperer
	ongoingTamperersUnderMutex  []OngoingTamper
}

func NewTamperingTransport() TamperingTransport {
	return &tamperingTransport{
		transportListenersUnderMutex: make(map[string]adapter.TransportListener),
		tampererLock:                 &sync.RWMutex{},
		listenerLock:                 &sync.RWMutex{},
	}
}

func (t *tamperingTransport) RegisterListener(listener adapter.TransportListener, listenerPublicKey primitives.Ed25519PublicKey) {
	t.listenerLock.Lock()
	defer t.listenerLock.Unlock()
	t.transportListenersUnderMutex[string(listenerPublicKey)] = listener
}

func (t *tamperingTransport) Send(ctx context.Context, data *adapter.TransportData) error {
	t.releaseLatches(data)

	if err, shouldReturn := t.maybeTamper(ctx, data); shouldReturn {
		return err
	}

	go t.receive(ctx, data)
	return nil
}

func (t *tamperingTransport) maybeTamper(ctx context.Context, data *adapter.TransportData) (error, bool) {
	t.tampererLock.RLock()
	defer t.tampererLock.RUnlock()
	for _, o := range t.ongoingTamperersUnderMutex {
		if err, shouldReturn := o.maybeTamper(ctx, data); shouldReturn {
			return err, shouldReturn
		}
	}

	return nil, false

}

func (t *tamperingTransport) Pause(predicate MessagePredicate) OngoingTamper {
	return t.addTamperer(&pausingTamperer{predicate: predicate, transport: t, lock: &sync.Mutex{}})
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

func (t *tamperingTransport) Delay(duration func() time.Duration, predicate MessagePredicate) OngoingTamper {
	return t.addTamperer(&delayingTamperer{predicate: predicate, transport: t, duration: duration})
}

func (t *tamperingTransport) LatchOn(predicate MessagePredicate) LatchingTamper {
	tamperer := &latchingTamperer{predicate: predicate, transport: t, cond: sync.NewCond(&sync.Mutex{})}
	t.tampererLock.Lock()
	defer t.tampererLock.Unlock()
	t.latchingTamperersUnderMutex = append(t.latchingTamperersUnderMutex, tamperer)

	tamperer.cond.L.Lock()
	return tamperer
}

func (t *tamperingTransport) removeOngoingTamperer(tamperer OngoingTamper) {
	t.tampererLock.Lock()
	defer t.tampererLock.Unlock()
	a := t.ongoingTamperersUnderMutex
	for p, v := range a {
		if v == tamperer {
			a[p] = a[len(a)-1]
			a[len(a)-1] = nil
			a = a[:len(a)-1]

			t.ongoingTamperersUnderMutex = a

			return
		}
	}
	panic("Tamperer not found in ongoing tamperer list")
}

func (t *tamperingTransport) removeLatchingTamperer(tamperer *latchingTamperer) {
	t.tampererLock.Lock()
	defer t.tampererLock.Unlock()
	a := t.latchingTamperersUnderMutex
	for p, v := range a {
		if v == tamperer {
			a[p] = a[len(a)-1]
			a[len(a)-1] = nil
			a = a[:len(a)-1]

			t.latchingTamperersUnderMutex = a

			return
		}
	}
	panic("Tamperer not found in ongoing tamperer list")
}

func (t *tamperingTransport) receive(ctx context.Context, data *adapter.TransportData) {
	switch data.RecipientMode {

	case gossipmessages.RECIPIENT_LIST_MODE_BROADCAST:
		for _, l := range t.getTransportListenersExceptPublicKeys(data.SenderPublicKey) {
			l.OnTransportMessageReceived(ctx, data.Payloads)
		}

	case gossipmessages.RECIPIENT_LIST_MODE_LIST:
		for _, l := range t.getTransportListenersByPublicKeys(data.RecipientPublicKeys) {
			l.OnTransportMessageReceived(ctx, data.Payloads)
		}

	case gossipmessages.RECIPIENT_LIST_MODE_ALL_BUT_LIST:
		panic("Not implemented")
	}

}

func (t *tamperingTransport) getTransportListenersExceptPublicKeys(exceptPublicKey primitives.Ed25519PublicKey) (listeners []adapter.TransportListener) {
	t.listenerLock.RLock()
	defer t.listenerLock.RUnlock()

	for stringPublicKey, l := range t.transportListenersUnderMutex {
		if stringPublicKey != string(exceptPublicKey) {
			listeners = append(listeners, l)
		}
	}

	return listeners
}

func (t *tamperingTransport) getTransportListenersByPublicKeys(publicKeys []primitives.Ed25519PublicKey) (listeners []adapter.TransportListener) {
	t.listenerLock.RLock()
	defer t.listenerLock.RUnlock()

	for _, recipientPublicKey := range publicKeys {
		stringPublicKey := string(recipientPublicKey)
		if listener, found := t.transportListenersUnderMutex[stringPublicKey]; found {
			listeners = append(listeners, listener)
		}
	}

	return listeners
}

func (t *tamperingTransport) releaseLatches(data *adapter.TransportData) {
	t.tampererLock.RLock()
	defer t.tampererLock.RUnlock()

	for _, o := range t.latchingTamperersUnderMutex {
		if o.predicate(data) {
			o.cond.Signal()
		}
	}
}

func (t *tamperingTransport) addTamperer(tamperer OngoingTamper) OngoingTamper {
	t.tampererLock.Lock()
	defer t.tampererLock.Unlock()
	t.ongoingTamperersUnderMutex = append(t.ongoingTamperersUnderMutex, tamperer)
	return tamperer
}

type failingTamperer struct {
	predicate MessagePredicate
	transport *tamperingTransport
}

func (o *failingTamperer) maybeTamper(ctx context.Context, data *adapter.TransportData) (error, bool) {
	if o.predicate(data) {
		return &adapter.ErrTransportFailed{data}, true
	}

	return nil, false
}

func (o *failingTamperer) Release(ctx context.Context) {
	o.transport.removeOngoingTamperer(o)
}

type duplicatingTamperer struct {
	predicate MessagePredicate
	transport *tamperingTransport
}

func (o *duplicatingTamperer) maybeTamper(ctx context.Context, data *adapter.TransportData) (error, bool) {
	if o.predicate(data) {
		go func() {
			time.Sleep(10 * time.Millisecond)
			o.transport.receive(ctx, data)
		}()
	}
	return nil, false
}

func (o *duplicatingTamperer) Release(ctx context.Context) {
	o.transport.removeOngoingTamperer(o)
}

type delayingTamperer struct {
	predicate MessagePredicate
	transport *tamperingTransport
	duration  func() time.Duration
}

func (o *delayingTamperer) maybeTamper(ctx context.Context, data *adapter.TransportData) (error, bool) {
	if o.predicate(data) {
		go func() {
			time.Sleep(o.duration())
			o.transport.receive(ctx, data)
		}()
		return nil, true
	}

	return nil, false
}

func (o *delayingTamperer) Release(ctx context.Context) {
	o.transport.removeOngoingTamperer(o)
}

type corruptingTamperer struct {
	predicate MessagePredicate
	transport *tamperingTransport
}

func (o *corruptingTamperer) maybeTamper(ctx context.Context, data *adapter.TransportData) (error, bool) {
	if o.predicate(data) {
		for i := 0; i < 10; i++ {
			x := rand.Intn(len(data.Payloads))
			y := rand.Intn(len(data.Payloads[x]))
			data.Payloads[x][y] = 0
		}
	}
	return nil, false
}

func (o *corruptingTamperer) Release(ctx context.Context) {
	o.transport.removeOngoingTamperer(o)
}

type pausingTamperer struct {
	predicate MessagePredicate
	transport *tamperingTransport
	messages  []*adapter.TransportData
	lock      *sync.Mutex
}

func (o *pausingTamperer) maybeTamper(ctx context.Context, data *adapter.TransportData) (error, bool) {
	if o.predicate(data) {
		o.lock.Lock()
		defer o.lock.Unlock()
		o.messages = append(o.messages, data)
		return nil, true
	}

	return nil, false
}

func (o *pausingTamperer) Release(ctx context.Context) {
	o.transport.removeOngoingTamperer(o)
	for _, message := range o.messages {
		o.transport.Send(ctx, message)
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
