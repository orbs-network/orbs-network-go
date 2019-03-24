// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package codec

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/pkg/errors"
)

func EncodeForwardedTransactions(header *gossipmessages.Header, message *gossipmessages.ForwardedTransactionsMessage) ([][]byte, error) {
	if message.Sender == nil {
		return nil, errors.New("missing Sender")
	}
	if len(message.SignedTransactions) == 0 {
		return nil, errors.New("missing SignedTransactions")
	}

	payloads := make([][]byte, 0, 2+len(message.SignedTransactions))
	payloads = append(payloads, header.Raw())
	payloads = append(payloads, message.Sender.Raw())
	for _, tx := range message.SignedTransactions {
		payloads = append(payloads, tx.Raw())
	}

	return payloads, nil
}

func DecodeForwardedTransactions(payloads [][]byte) (*gossipmessages.ForwardedTransactionsMessage, error) {
	if len(payloads) < 2 {
		return nil, errors.New("wrong num of payloads")
	}

	txs := make([]*protocol.SignedTransaction, 0, len(payloads)-1)

	senderSignature := gossipmessages.SenderSignatureReader(payloads[0])
	if !senderSignature.IsValid() {
		return nil, errors.New("SenderSignature is corrupted and cannot be decoded")
	}
	for _, payload := range payloads[1:] {
		tx := protocol.SignedTransactionReader(payload)
		if !tx.IsValid() {
			return nil, errors.New("SignedTransaction is corrupted and cannot be decoded")
		}
		txs = append(txs, tx)
	}

	return &gossipmessages.ForwardedTransactionsMessage{
		Sender:             senderSignature,
		SignedTransactions: txs,
	}, nil
}
