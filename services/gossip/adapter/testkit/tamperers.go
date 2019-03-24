// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package testkit

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"runtime"
	"sync"
	"time"
)

type failingTamperer struct {
	predicate MessagePredicate
	transport *TamperingTransport
}

func (o *failingTamperer) maybeTamper(ctx context.Context, data *adapter.TransportData) (err error, returnWithoutSending bool) {
	if o.predicate(data) {
		return &adapter.ErrTransportFailed{Data: data}, true
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

func (o *duplicatingTamperer) maybeTamper(ctx context.Context, data *adapter.TransportData) (err error, returnWithoutSending bool) {
	if o.predicate(data) {
		supervised.GoOnce(o.transport.logger, func() {
			time.Sleep(10 * time.Millisecond)
			o.transport.sendToPeers(ctx, data)
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

func (o *delayingTamperer) maybeTamper(ctx context.Context, data *adapter.TransportData) (error, bool) {
	if o.predicate(data) {
		supervised.GoOnce(o.transport.logger, func() {
			time.Sleep(o.duration())
			o.transport.sendToPeers(ctx, data)
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

func (o *corruptingTamperer) maybeTamper(ctx context.Context, data *adapter.TransportData) (error, bool) {
	if o.predicate(data) {
		for i := 0; i < 10; i++ {
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

type pausingTamperer struct {
	predicate MessagePredicate
	transport *TamperingTransport
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

func (o *pausingTamperer) StopTampering(ctx context.Context) {
	o.transport.removeOngoingTamperer(o)
	for _, message := range o.messages {
		o.transport.Send(ctx, message)
		runtime.Gosched() // TODO(v1): this is required or else messages arrive in the opposite order after resume (supposedly fixed now when we moved to channels in transport)
	}
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
