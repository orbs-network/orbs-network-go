package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/crypto/signature"
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
		validateSignerAndContractName,
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
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_UNSUPPORTED_VERSION}
	}
	return nil
}

func validateSignerAndContractName(transaction *protocol.SignedTransaction) *ErrTransactionRejected {
	tx := transaction.Transaction()
	if tx.ContractName() == "" ||
		!tx.Signer().IsSchemeEddsa() ||
		len(tx.Signer().Eddsa().SignerPublicKey()) != signature.ED25519_PUBLIC_KEY_SIZE_BYTES {
		//TODO is this the correct status?
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH}
	}
	return nil
}

func validateTransactionNotExpired(vctx *validationContext) validator {
	return func(transaction *protocol.SignedTransaction) *ErrTransactionRejected {
		if transaction.Transaction().Timestamp() < primitives.TimestampNano(time.Now().Add(vctx.expiryWindow*-1).UnixNano()) {
			return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED}
		}

		return nil
	}
}

func validateTransactionNotInFuture(vctx *validationContext) validator {
	return func(transaction *protocol.SignedTransaction) *ErrTransactionRejected {
		tsWithGrace := primitives.TimestampNano(time.Now().UnixNano() + vctx.futureTimestampGrace.Nanoseconds())
		if transaction.Transaction().Timestamp() > tsWithGrace {
			return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED}
		}

		return nil
	}
}

func validateTransactionVirtualChainId(vctx *validationContext) validator {
	return func(transaction *protocol.SignedTransaction) *ErrTransactionRejected {
		if !transaction.Transaction().VirtualChainId().Equal(vctx.virtualChainId) {
			return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_MISMATCH}

		}
		return nil
	}
}
