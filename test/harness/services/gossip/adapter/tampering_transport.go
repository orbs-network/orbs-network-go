package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"sync"
	"time"
)

// The TamperingTransport is an in-memory implementation of the Gossip Transport adapter, that adds the ability
// to tamper with the messages or to synchronize the test's goroutine with the SUT's goroutines
type Tamperer interface {

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
	Corrupt(predicate MessagePredicate, rand *test.ControlledRand) OngoingTamper

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

type TamperingTransport struct {
	nested adapter.Transport

	tamperers struct {
		sync.RWMutex
		latches          []*latchingTamperer
		ongoingTamperers []OngoingTamper
	}

	logger log.BasicLogger
}

func NewTamperingTransport(logger log.BasicLogger, nested adapter.Transport) *TamperingTransport {
	t := &TamperingTransport{
		logger: logger.WithTags(log.String("adapter", "transport")),
		nested: nested,
	}

	return t
}

func (t *TamperingTransport) RegisterListener(listener adapter.TransportListener, listenerNodeAddress primitives.NodeAddress) {
	t.nested.RegisterListener(listener, listenerNodeAddress)
}

func (t *TamperingTransport) Send(ctx context.Context, data *adapter.TransportData) error {
	t.releaseLatches(data)

	if err, shouldReturn := t.maybeTamper(ctx, data); shouldReturn {
		return err
	}

	supervised.GoOnce(t.logger, func() {
		t.sendToPeers(ctx, data)
	})

	return nil
}

func (t *TamperingTransport) maybeTamper(ctx context.Context, data *adapter.TransportData) (error, bool) {
	t.tamperers.RLock()
	defer t.tamperers.RUnlock()
	for _, o := range t.tamperers.ongoingTamperers {
		if err, shouldReturn := o.maybeTamper(ctx, data); shouldReturn {
			return err, shouldReturn
		}
	}
	return nil, false
}

func (t *TamperingTransport) Pause(predicate MessagePredicate) OngoingTamper {
	return t.addTamperer(&pausingTamperer{predicate: predicate, transport: t, lock: &sync.Mutex{}})
}

func (t *TamperingTransport) Fail(predicate MessagePredicate) OngoingTamper {
	return t.addTamperer(&failingTamperer{predicate: predicate, transport: t})
}

func (t *TamperingTransport) Duplicate(predicate MessagePredicate) OngoingTamper {
	return t.addTamperer(&duplicatingTamperer{predicate: predicate, transport: t})
}

func (t *TamperingTransport) Corrupt(predicate MessagePredicate, ctrlRand *test.ControlledRand) OngoingTamper {
	return t.addTamperer(&corruptingTamperer{
		predicate: predicate,
		transport: t,
		ctrlRand:  ctrlRand,
	})
}

func (t *TamperingTransport) Delay(duration func() time.Duration, predicate MessagePredicate) OngoingTamper {
	return t.addTamperer(&delayingTamperer{predicate: predicate, transport: t, duration: duration})
}

func (t *TamperingTransport) LatchOn(predicate MessagePredicate) LatchingTamper {
	tamperer := &latchingTamperer{predicate: predicate, transport: t, cond: sync.NewCond(&sync.Mutex{})}
	t.tamperers.Lock()
	defer t.tamperers.Unlock()
	t.tamperers.latches = append(t.tamperers.latches, tamperer)

	tamperer.cond.L.Lock()
	return tamperer
}

func (t *TamperingTransport) removeOngoingTamperer(tamperer OngoingTamper) {
	t.tamperers.Lock()
	defer t.tamperers.Unlock()
	a := t.tamperers.ongoingTamperers
	for p, v := range a {
		if v == tamperer {
			a[p] = a[len(a)-1]
			a[len(a)-1] = nil
			a = a[:len(a)-1]

			t.tamperers.ongoingTamperers = a

			return
		}
	}
	panic("Tamperer not found in ongoing tamperer list")
}

func (t *TamperingTransport) removeLatchingTamperer(tamperer *latchingTamperer) {
	t.tamperers.Lock()
	defer t.tamperers.Unlock()
	a := t.tamperers.latches
	for p, v := range a {
		if v == tamperer {
			a[p] = a[len(a)-1]
			a[len(a)-1] = nil
			a = a[:len(a)-1]

			t.tamperers.latches = a

			return
		}
	}
	panic("Tamperer not found in ongoing tamperer list")
}

func (t *TamperingTransport) sendToPeers(ctx context.Context, data *adapter.TransportData) {
	t.nested.Send(ctx, data)
}

func (t *TamperingTransport) releaseLatches(data *adapter.TransportData) {
	t.tamperers.RLock()
	defer t.tamperers.RUnlock()

	for _, o := range t.tamperers.latches {
		if o.predicate(data) {
			o.cond.Signal()
		}
	}
}

func (t *TamperingTransport) addTamperer(tamperer OngoingTamper) OngoingTamper {
	t.tamperers.Lock()
	defer t.tamperers.Unlock()
	t.tamperers.ongoingTamperers = append(t.tamperers.ongoingTamperers, tamperer)
	return tamperer
}
