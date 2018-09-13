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
}

// A predicate for matching messages with a certain property
type MessagePredicate func(data *adapter.TransportData) bool

type OngoingTamper interface {
	Release()
}

type LatchingTamper interface {
	Wait()
	Remove()
}

type tamperingTransport struct {
	mutex                *sync.Mutex
	transportListeners   map[string]adapter.TransportListener
	failingTamperers     []*failingTamperer
	pausingTamperers     []*pausingTamperer
	latchingTamperers    []*latchingTamperer
	duplicatingTamperers []*duplicatingTamperer
	corruptingTamperer   []*corruptingTamperer
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

	if t.shouldFail(data) {
		return &adapter.ErrTransportFailed{data}
	}

	if t.hasPaused(data) {
		return nil
	}

	if t.shouldDuplicate(data) {
		go func() {
			time.Sleep(10 * time.Millisecond)
			t.receive(data)
		}()
	}

	if t.shouldCorrupt(data) {
		for i := 0; i < 10; i++ {
			x := rand.Intn(len(data.Payloads))
			y := rand.Intn(len(data.Payloads[x]))
			data.Payloads[x][y] = 0
		}
	}

	go t.receive(data)
	return nil
}

func (t *tamperingTransport) Pause(predicate MessagePredicate) OngoingTamper {
	tamperer := &pausingTamperer{predicate: predicate, transport: t}
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.pausingTamperers = append(t.pausingTamperers, tamperer)
	return tamperer
}

func (t *tamperingTransport) Fail(predicate MessagePredicate) OngoingTamper {
	tamperer := &failingTamperer{predicate: predicate, transport: t}
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.failingTamperers = append(t.failingTamperers, tamperer)
	return tamperer
}

func (t *tamperingTransport) Duplicate(predicate MessagePredicate) OngoingTamper {
	tamperer := &duplicatingTamperer{predicate: predicate, transport: t}
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.duplicatingTamperers = append(t.duplicatingTamperers, tamperer)
	return tamperer
}

func (t *tamperingTransport) Corrupt(predicate MessagePredicate) OngoingTamper {
	tamperer := &corruptingTamperer{predicate: predicate, transport: t}
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.corruptingTamperer = append(t.corruptingTamperer, tamperer)
	return tamperer
}

func (t *tamperingTransport) LatchOn(predicate MessagePredicate) LatchingTamper {
	tamperer := &latchingTamperer{predicate: predicate, transport: t, cond: sync.NewCond(&sync.Mutex{})}
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.latchingTamperers = append(t.latchingTamperers, tamperer)

	tamperer.cond.L.Lock()
	return tamperer
}

func (t *tamperingTransport) removeFailTamperer(tamperer *failingTamperer) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	a := t.failingTamperers
	for p, v := range a {
		if v == tamperer {
			a[p] = a[len(a)-1]
			a[len(a)-1] = nil
			a = a[:len(a)-1]

			t.failingTamperers = a

			return
		}
	}
	panic("Tamperer not found in ongoing tamperer list")
}

func (t *tamperingTransport) removeDuplicatingTamperer(tamperer *duplicatingTamperer) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	a := t.duplicatingTamperers
	for p, v := range a {
		if v == tamperer {
			a[p] = a[len(a)-1]
			a[len(a)-1] = nil
			a = a[:len(a)-1]

			t.duplicatingTamperers = a

			return
		}
	}
	panic("Tamperer not found in ongoing tamperer list")
}

func (t *tamperingTransport) removeCorruptingTamperer(tamperer *corruptingTamperer) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	a := t.corruptingTamperer
	for p, v := range a {
		if v == tamperer {
			a[p] = a[len(a)-1]
			a[len(a)-1] = nil
			a = a[:len(a)-1]

			t.corruptingTamperer = a

			return
		}
	}
	panic("Tamperer not found in ongoing tamperer list")
}

func (t *tamperingTransport) removePauseTamperer(tamperer *pausingTamperer) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	a := t.pausingTamperers
	for p, v := range a {
		if v == tamperer {
			a[p] = a[len(a)-1]
			a[len(a)-1] = nil
			a = a[:len(a)-1]

			t.pausingTamperers = a

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

func (t *tamperingTransport) shouldFail(data *adapter.TransportData) bool {
	for _, o := range t.failingTamperers {
		if o.predicate(data) {
			return true
		}
	}
	return false
}

func (t *tamperingTransport) shouldDuplicate(data *adapter.TransportData) bool {
	for _, o := range t.duplicatingTamperers {
		if o.predicate(data) {
			return true
		}
	}
	return false
}

func (t *tamperingTransport) shouldCorrupt(data *adapter.TransportData) bool {
	for _, o := range t.corruptingTamperer {
		if o.predicate(data) {
			return true
		}
	}
	return false
}

func (t *tamperingTransport) hasPaused(data *adapter.TransportData) bool {
	for _, p := range t.pausingTamperers {
		if p.predicate(data) {
			t.mutex.Lock()
			defer t.mutex.Unlock()
			p.messages = append(p.messages, data)
			return true
		}
	}
	return false
}
func (t *tamperingTransport) releaseLatches(data *adapter.TransportData) {
	for _, o := range t.latchingTamperers {
		if o.predicate(data) {
			o.cond.Signal()
		}
	}
}

type failingTamperer struct {
	predicate MessagePredicate
	transport *tamperingTransport
}

type duplicatingTamperer struct {
	predicate MessagePredicate
	transport *tamperingTransport
}

type corruptingTamperer struct {
	predicate MessagePredicate
	transport *tamperingTransport
}

type pausingTamperer struct {
	predicate MessagePredicate
	transport *tamperingTransport
	messages  []*adapter.TransportData
}

type latchingTamperer struct {
	predicate MessagePredicate
	transport *tamperingTransport
	cond      *sync.Cond
}

func (o *failingTamperer) Release() {
	o.transport.removeFailTamperer(o)
}

func (o *duplicatingTamperer) Release() {
	o.transport.removeDuplicatingTamperer(o)
}

func (o *corruptingTamperer) Release() {
	o.transport.removeCorruptingTamperer(o)
}

func (o *pausingTamperer) Release() {
	o.transport.removePauseTamperer(o)
	for _, message := range o.messages {
		o.transport.Send(message)
		runtime.Gosched() // TODO: this is required or else messages arrive in the opposite order after resume
	}
}

func (o *latchingTamperer) Remove() {
	o.transport.removeLatchingTamperer(o)
}

func (o *latchingTamperer) Wait() {
	o.cond.Wait() // TODO: change cond to channel close
}
