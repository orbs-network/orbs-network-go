// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package testkit

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"sync"
	"time"
)

type failingTamperer struct {
	predicate MessagePredicate
	transport *TamperingTransport
}

type messageDroppedByTamperer struct {
	Data *adapter.TransportData
}

func (e *messageDroppedByTamperer) Error() string {
	return fmt.Sprintf("tampering transport intentionally failed to send: %v", e.Data)
}

func (o *failingTamperer) maybeTamper(ctx context.Context, data *adapter.TransportData, peerAddress primitives.NodeAddress, transmit adapter.TransmitFunc) (err error, returnWithoutSending bool) {
	if o.predicate(data) {
		return &messageDroppedByTamperer{Data: data}, true
	}

	return nil, false
}

func (o *failingTamperer) StopTampering(ctx context.Context) {
	o.transport.removeOngoingTamperer(o)
}

type duplicatingTamperer struct {
	predicate MessagePredicate
	transport *TamperingTransport
}

func (o *duplicatingTamperer) maybeTamper(ctx context.Context, data *adapter.TransportData, peerAddress primitives.NodeAddress, transmit adapter.TransmitFunc) (err error, returnWithoutSending bool) {
	if o.predicate(data) {
		govnr.Once(logfields.GovnrErrorer(o.transport.logger), func() {
			time.Sleep(10 * time.Millisecond)
			transmit(ctx, peerAddress, data)
		})
	}
	return nil, false
}

func (o *duplicatingTamperer) StopTampering(ctx context.Context) {
	o.transport.removeOngoingTamperer(o)
}

type delayingTamperer struct {
	predicate MessagePredicate
	transport *TamperingTransport
	duration  func() time.Duration
}

func (o *delayingTamperer) maybeTamper(ctx context.Context, data *adapter.TransportData, peerAddress primitives.NodeAddress, transmit adapter.TransmitFunc) (error, bool) {
	if o.predicate(data) {
		govnr.Once(logfields.GovnrErrorer(o.transport.logger), func() {
			time.Sleep(o.duration())
			transmit(ctx, peerAddress, data)
		})
		return nil, true
	}

	return nil, false
}

func (o *delayingTamperer) StopTampering(ctx context.Context) {
	o.transport.removeOngoingTamperer(o)
}

type corruptingTamperer struct {
	predicate MessagePredicate
	transport *TamperingTransport
	ctrlRand  *rand.ControlledRand
}

func (o *corruptingTamperer) maybeTamper(ctx context.Context, data *adapter.TransportData, peerAddress primitives.NodeAddress, transmit adapter.TransmitFunc) (error, bool) {
	if o.predicate(data) {
		// An odd iteration count will ensure at least one corruption
		for i := 0; i < 11; i++ {
			if len(data.Payloads) == 0 {
				continue
			}
			x := o.ctrlRand.Intn(len(data.Payloads))
			if len(data.Payloads[x]) == 0 {
				continue
			}
			y := o.ctrlRand.Intn(len(data.Payloads[x]))
			data.Payloads[x][y] ^= 0x55 // 0x55 is 01010101 so XORing with it reverses all bits on the byte - this actually does something even if original byte was 0
		}
	}
	return nil, false
}

func (o *corruptingTamperer) StopTampering(ctx context.Context) {
	o.transport.removeOngoingTamperer(o)
}

type pausedTransmission struct {
	data        *adapter.TransportData
	peerAddress primitives.NodeAddress
	transmit    adapter.TransmitFunc
}

type pausingTamperer struct {
	predicate           MessagePredicate
	transport           *TamperingTransport
	pausedTransmissions []*pausedTransmission
	lock                *sync.Mutex
}

func (o *pausingTamperer) maybeTamper(ctx context.Context, data *adapter.TransportData, peerAddress primitives.NodeAddress, transmit adapter.TransmitFunc) (error, bool) {
	if o.predicate(data) {
		o.lock.Lock()
		defer o.lock.Unlock()
		o.pausedTransmissions = append(o.pausedTransmissions, &pausedTransmission{
			data:        data,
			peerAddress: peerAddress,
			transmit:    transmit,
		})
		return nil, true
	}

	return nil, false
}

func (o *pausingTamperer) StopTampering(ctx context.Context) {
	o.transport.removeOngoingTamperer(o)
	for _, t := range o.pausedTransmissions {
		t.transmit(ctx, t.peerAddress, t.data)
	}
	o.pausedTransmissions = o.pausedTransmissions[:0]
}

type latchingTamperer struct {
	predicate MessagePredicate
	transport *TamperingTransport
	cond      *sync.Cond
}

func (o *latchingTamperer) Remove() {
	o.transport.removeLatchingTamperer(o)
}

func (o *latchingTamperer) Wait() {
	o.cond.Wait()
}
