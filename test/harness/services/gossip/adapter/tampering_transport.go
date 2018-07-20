package adapter

import (
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"runtime"
	"sync"
)

type MessagePredicate func(data *adapter.TransportData) bool

type OngoingTamper interface {
	Release()
}

type LatchingTamper interface {
	Wait()
	Remove()
}

type TamperingTransport interface {
	adapter.Transport

	Fail(predicate MessagePredicate) OngoingTamper
	Pause(predicate MessagePredicate) OngoingTamper
	LatchOn(predicate MessagePredicate) LatchingTamper
}

type tamperingTransport struct {
	transportListeners map[string]adapter.TransportListener

	mutex             *sync.Mutex
	failingTamperers  []*failingTamperer
	pausingTamperers  []*pausingTamperer
	latchingTamperers []*latchingTamperer
}

func NewTamperingTransport() TamperingTransport {
	return &tamperingTransport{
		transportListeners: make(map[string]adapter.TransportListener),
		mutex:              &sync.Mutex{},
	}
}

func (t *tamperingTransport) RegisterListener(listener adapter.TransportListener, listenerPublicKey primitives.Ed25519Pkey) {
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
		for stringPublicKey, l := range t.transportListeners {
			if stringPublicKey != string(data.SenderPublicKey) {
				l.OnTransportMessageReceived(data.Payloads)
			}
		}

	case gossipmessages.RECIPIENT_LIST_MODE_LIST:
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
	o.cond.Wait()
}

