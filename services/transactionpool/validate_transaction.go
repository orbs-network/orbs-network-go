// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/scribe/log"
	"time"
)

type validationContext struct {
	nodeSyncRejectInterval time.Duration
	expiryWindow           time.Duration
	futureTimestampGrace   time.Duration
	virtualChainId         primitives.VirtualChainId
}

func (c *validationContext) ValidateAddedTransaction(transaction *protocol.SignedTransaction, currentTime time.Time, lastCommittedBlockTimestamp primitives.TimestampNano) *ErrTransactionRejected {
	proposedBlockTimestamp := primitives.TimestampNano(currentTime.UnixNano())

	if err := c.validateAddTransactionProtocolVersion(transaction); err != nil {
		return err
	}
	if err := c.validateSignatureType(transaction); err != nil {
		return err
	}
	if err := c.validateTransactionVirtualChainId(transaction); err != nil {
		return err
	}
	if err := c.validateNodeIsInSync(currentTime, lastCommittedBlockTimestamp); err != nil {
		return err
	}
	if err := c.validateTransactionNotExpired(transaction, proposedBlockTimestamp); err != nil {
		return err
	}
	if err := c.validateTransactionNotInFuture(transaction, proposedBlockTimestamp); err != nil {
		return err
	}
	return nil
}

func (c *validationContext) ValidateTransactionForOrdering(transaction *protocol.SignedTransaction, proposedBlockProtocolVersion primitives.ProtocolVersion, proposedBlockTimestamp primitives.TimestampNano) *ErrTransactionRejected {
	if err := c.validateProtocolVersion(transaction, proposedBlockProtocolVersion); err != nil {
		return err
	}
	if err := c.validateSignatureType(transaction); err != nil {
		return err
	}
	if err := c.validateTransactionVirtualChainId(transaction); err != nil {
		return err
	}
	if err := c.validateTransactionNotExpired(transaction, proposedBlockTimestamp); err != nil {
		return err
	}
	if err := c.validateTransactionNotInFuture(transaction, proposedBlockTimestamp); err != nil {
		return err
	}
	return nil
}

func (c *validationContext) validateAddTransactionProtocolVersion(transaction *protocol.SignedTransaction) *ErrTransactionRejected {
	if transaction.Transaction().ProtocolVersion() > config.MAXIMAL_PROTOCOL_VERSION_SUPPORTED_VALUE {
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_UNSUPPORTED_VERSION, log.Stringable("maximal-protocol-version", config.MAXIMAL_PROTOCOL_VERSION_SUPPORTED_VALUE), log.Stringable("protocol-version", transaction.Transaction().ProtocolVersion())}
	}
	return nil
}

func (c *validationContext) validateProtocolVersion(transaction *protocol.SignedTransaction, proposedBlockProtocolVersion primitives.ProtocolVersion) *ErrTransactionRejected {
	if transaction.Transaction().ProtocolVersion() > proposedBlockProtocolVersion {
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_UNSUPPORTED_VERSION, log.Stringable("block-protocol-version", proposedBlockProtocolVersion), log.Stringable("protocol-version", transaction.Transaction().ProtocolVersion())}
	}
	return nil
}

func (c *validationContext) validateTransactionVirtualChainId(transaction *protocol.SignedTransaction) *ErrTransactionRejected {
	if !transaction.Transaction().VirtualChainId().Equal(c.virtualChainId) {
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_MISMATCH, logfields.VirtualChainId(c.virtualChainId), logfields.VirtualChainId(transaction.Transaction().VirtualChainId())}
	}
	return nil
}

func (c *validationContext) validateSignatureType(transaction *protocol.SignedTransaction) *ErrTransactionRejected {
	tx := transaction.Transaction()
	if !tx.Signer().IsSchemeEddsa() {
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_UNKNOWN_SIGNER_SCHEME, log.String("signer-scheme", "Eddsa"), log.Stringable("signer", tx.Signer())}
	}

	if len(tx.Signer().Eddsa().SignerPublicKey()) != keys.ED25519_PUBLIC_KEY_SIZE_BYTES {
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH, log.Int("signature-length", keys.ED25519_PUBLIC_KEY_SIZE_BYTES), log.Int("signature-length", len(tx.Signer().Eddsa().SignerPublicKey()))}
	}

	return nil
}

func (c *validationContext) validateNodeIsInSync(currentTime time.Time, lastCommittedBlockTimestamp primitives.TimestampNano) *ErrTransactionRejected {
	threshold := primitives.TimestampNano(currentTime.Add(c.nodeSyncRejectInterval * -1).UnixNano())
	if lastCommittedBlockTimestamp < threshold {
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_NODE_OUT_OF_SYNC, logfields.TimestampNano("min-timestamp", threshold), logfields.TimestampNano("last-committed-block-timestamp", lastCommittedBlockTimestamp)}
	}
	return nil
}

func (c *validationContext) validateTransactionNotExpired(transaction *protocol.SignedTransaction, proposedBlockTimestamp primitives.TimestampNano) *ErrTransactionRejected {
	threshold := proposedBlockTimestamp - primitives.TimestampNano(c.expiryWindow.Nanoseconds())
	if transaction.Transaction().Timestamp() < threshold {
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED, logfields.TimestampNano("min-timestamp", threshold), logfields.TimestampNano("tx-timestamp", transaction.Transaction().Timestamp())}
	}
	return nil
}

func (c *validationContext) validateTransactionNotInFuture(transaction *protocol.SignedTransaction, proposedBlockTimestamp primitives.TimestampNano) *ErrTransactionRejected {
	tsWithGrace := proposedBlockTimestamp + primitives.TimestampNano(c.futureTimestampGrace.Nanoseconds())
	if transaction.Transaction().Timestamp() > tsWithGrace {
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_AHEAD_OF_NODE_TIME, logfields.TimestampNano("max-timestamp", tsWithGrace), logfields.TimestampNano("tx-timestamp", transaction.Transaction().Timestamp())}
	}
	return nil
}
