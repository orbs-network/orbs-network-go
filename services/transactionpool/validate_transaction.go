package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"time"
)

const ProtocolVersion = primitives.ProtocolVersion(1)

type validator func(transaction *protocol.SignedTransaction) *ErrTransactionRejected

type validationContext struct {
	expiryWindow                time.Duration
	lastCommittedBlockTimestamp primitives.TimestampNano
	futureTimestampGrace        time.Duration
	virtualChainId              primitives.VirtualChainId
}

func (c *validationContext) validateTransaction(transaction *protocol.SignedTransaction) *ErrTransactionRejected {
	//TODO can we create the list of validators once on system startup; this will save on performance in the critical path
	validators := []validator{
		validateProtocolVersion,
		validateContractName,
		validateSignature,
		validateTransactionNotExpired(c),
		validateTransactionNotInFuture(c),
		validateTransactionVirtualChainId(c),
	}

	for _, validate := range validators {
		err := validate(transaction)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateProtocolVersion(tx *protocol.SignedTransaction) *ErrTransactionRejected {
	if tx.Transaction().ProtocolVersion() != ProtocolVersion {
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_UNSUPPORTED_VERSION, log.Stringable("protocol-version", ProtocolVersion), log.Stringable("protocol-version", tx.Transaction().ProtocolVersion())}
	}
	return nil
}

func validateSignature(transaction *protocol.SignedTransaction) *ErrTransactionRejected {
	tx := transaction.Transaction()
	if !tx.Signer().IsSchemeEddsa() {
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_UNKNOWN_SIGNER_SCHEME, log.String("signer-scheme", "Eddsa"), log.Stringable("signer", tx.Signer())}
	}

	if len(tx.Signer().Eddsa().SignerPublicKey()) != signature.ED25519_PUBLIC_KEY_SIZE_BYTES {
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH, log.Int("signature-length", signature.ED25519_PUBLIC_KEY_SIZE_BYTES), log.Int("signature-length", len(tx.Signer().Eddsa().SignerPublicKey()))}
	}

	//TODO actually verify the signature

	return nil
}

func validateContractName(transaction *protocol.SignedTransaction) *ErrTransactionRejected {
	tx := transaction.Transaction()
	if tx.ContractName() == "" {
		//TODO what is the expected status?
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_RESERVED, log.String("contract-name", "non-empty"), log.String("contract-name", "")}
	}

	return nil
}

func validateTransactionNotExpired(vctx *validationContext) validator {
	return func(transaction *protocol.SignedTransaction) *ErrTransactionRejected {
		threshold := primitives.TimestampNano(time.Now().Add(vctx.expiryWindow * -1).UnixNano())
		if transaction.Transaction().Timestamp() < threshold {
			return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED, log.TimestampNano("min-timestamp", threshold), log.TimestampNano("tx-timestamp", transaction.Transaction().Timestamp())}
		}

		return nil
	}
}

func validateTransactionNotInFuture(vctx *validationContext) validator {
	return func(transaction *protocol.SignedTransaction) *ErrTransactionRejected {
		tsWithGrace := vctx.lastCommittedBlockTimestamp + primitives.TimestampNano(vctx.futureTimestampGrace.Nanoseconds())
		if transaction.Transaction().Timestamp() > tsWithGrace {
			return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_AHEAD_OF_NODE_TIME, log.TimestampNano("max-timestamp", tsWithGrace), log.TimestampNano("tx-timestamp", transaction.Transaction().Timestamp())}
		}

		return nil
	}
}

func validateTransactionVirtualChainId(vctx *validationContext) validator {
	return func(transaction *protocol.SignedTransaction) *ErrTransactionRejected {
		if !transaction.Transaction().VirtualChainId().Equal(vctx.virtualChainId) {
			return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_MISMATCH, log.VirtualChainId(vctx.virtualChainId), log.VirtualChainId(transaction.Transaction().VirtualChainId())}

		}
		return nil
	}
}
