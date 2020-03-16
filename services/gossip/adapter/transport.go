// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

type TransportData struct {
	SenderNodeAddress      primitives.NodeAddress
	RecipientMode          gossipmessages.RecipientsListMode
	RecipientNodeAddresses []primitives.NodeAddress
	Payloads               [][]byte // the first payload is normally gossipmessages.Header
}

type Transport interface {
	supervised.GracefulShutdowner
	govnr.ShutdownWaiter
	RegisterListener(listener TransportListener, listenerNodeAddress primitives.NodeAddress)
	Send(ctx context.Context, data *TransportData) error // TODO don't return error. misleading meaning. use panics instead
	UpdateTopology(bgCtx context.Context, newPeers GossipPeers)
}

type TransportListener interface {
	fmt.Stringer // TODO smelly
	OnTransportMessageReceived(ctx context.Context, payloads [][]byte)
}

func (d *TransportData) TotalSize() (res int) {
	for _, payload := range d.Payloads {
		res += len(payload)
	}
	return
}

func (d *TransportData) Clone() *TransportData {
	var payloads [][]byte
	if d.Payloads != nil {
		payloads = make([][]byte, len(d.Payloads))
		for i, payload := range d.Payloads {
			payloads[i] = append(payload[:0:0], payload...)
		}
	}
	return &TransportData{
		SenderNodeAddress:      d.SenderNodeAddress,
		RecipientMode:          d.RecipientMode,
		RecipientNodeAddresses: append(d.RecipientNodeAddresses[:0:0], d.RecipientNodeAddresses...),
		Payloads:               payloads,
	}
}
