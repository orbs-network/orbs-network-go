package adapter

import (
	"context"
	"fmt"
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
	RegisterListener(listener TransportListener, listenerNodeAddress primitives.NodeAddress)
	Send(ctx context.Context, data *TransportData) error
}

type TransportListener interface {
	OnTransportMessageReceived(ctx context.Context, payloads [][]byte)
}

type ErrCorruptData struct {
}

func (e *ErrCorruptData) Error() string {
	return fmt.Sprintf("transport data is corrupt and missing required fields")
}

type ErrTransportFailed struct {
	Data *TransportData
}

func (e *ErrTransportFailed) Error() string {
	return fmt.Sprintf("transport failed to send: %v", e.Data)
}

func (d *TransportData) TotalSize() (res int) {
	for _, payload := range d.Payloads {
		res += len(payload)
	}
	return
}
