package transactionpool

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"time"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

const ProtocolVersion = primitives.ProtocolVersion(1)

type validator func(transaction *protocol.SignedTransaction) error

func validateTransaction(transaction *protocol.SignedTransaction) error {
	validators := []validator {
		validateTimestamp,
		validateProtocolVersion,
		validateSenderAndContractAddress,
	}

	for _, validate := range validators {
		err := validate(transaction)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateProtocolVersion(tx *protocol.SignedTransaction) error {
	if tx.Transaction().ProtocolVersion() != ProtocolVersion {
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_UNSUPPORTED_VERSION}
	}
	return nil
}

func validateTimestamp(transaction *protocol.SignedTransaction) error {
	if uint64(transaction.Transaction().Timestamp()) > uint64(time.Now().UnixNano()) {
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_TIME_STAMP_WINDOW_EXCEEDED}
	}
	return nil
}

func validateSenderAndContractAddress(transaction *protocol.SignedTransaction) error {
	tx := transaction.Transaction()
	if tx.ContractName() == "" || !tx.Signer().IsValid() || len(tx.Signer().Eddsa().SignerPublicKey()) == 0 {
		//TODO is this the correct status?
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH}
	}
	return nil
}


