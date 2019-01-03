package consensuscontext

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/merkle"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

func calculateTransactionsMerkleRoot(txs []*protocol.SignedTransaction) (primitives.Sha256, error) {
	txHashValues := make([]primitives.Sha256, len(txs))
	for i := 0; i < len(txs); i++ {
		txHashValues[i] = digest.CalcTxHash(txs[i].Transaction())
	}
	return merkle.CalculateOrderedTreeRoot(txHashValues), nil
}

func calculateReceiptsMerkleRoot(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error) {
	rptHashValues := make([]primitives.Sha256, len(receipts))
	for i := 0; i < len(receipts); i++ {
		rptHashValues[i] = digest.CalcReceiptHash(receipts[i])
	}
	return merkle.CalculateOrderedTreeRoot(rptHashValues), nil
}

func calculateStateDiffMerkleRoot(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error) {
	stdHashValues := make([]primitives.Sha256, len(stateDiffs))
	for i := 0; i < len(stateDiffs); i++ {
		stdHashValues[i] = digest.CalcContractStateDiffHash(stateDiffs[i])
	}
	return merkle.CalculateOrderedTreeRoot(stdHashValues), nil
}

func calculateNewBlockTimestamp(prevBlockTimestamp primitives.TimestampNano, now primitives.TimestampNano) primitives.TimestampNano {
	if now > prevBlockTimestamp {
		return now + 1
	}
	return prevBlockTimestamp + 1
}
