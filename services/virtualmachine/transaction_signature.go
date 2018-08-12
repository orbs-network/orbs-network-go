package virtualmachine

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

func (s *service) verifyTransactionSignatures(signedTransactions []*protocol.SignedTransaction, resultStatuses []protocol.TransactionStatus) (err error) {
	err = nil

	for i, signedTransaction := range signedTransactions {
		switch signedTransaction.Transaction().Signer().Scheme() {
		case protocol.SIGNER_SCHEME_EDDSA:
			if verifyEd25519Signer(signedTransaction) {
				resultStatuses[i] = protocol.TRANSACTION_STATUS_PRE_ORDER_VALID
			} else {
				resultStatuses[i] = protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH
				err = errors.New("not all transactions passed signature verification")
			}
		default:
			resultStatuses[i] = protocol.TRANSACTION_STATUS_REJECTED_UNKNOWN_SIGNER_SCHEME
			err = errors.New("not all transactions passed signature verification")
		}
	}

	return err
}

func verifyEd25519Signer(signedTransaction *protocol.SignedTransaction) bool {
	signerPublicKey := signedTransaction.Transaction().Signer().Eddsa().SignerPublicKey()
	txHash := digest.CalcTxHash(signedTransaction.Transaction())
	return signature.VerifyEd25519(signerPublicKey, txHash, signedTransaction.Signature())
}
