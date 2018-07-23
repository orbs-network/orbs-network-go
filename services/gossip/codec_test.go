package gossip

import (
	"github.com/google/go-cmp/cmp"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"testing"
)

var blockPairTable = []struct {
	origin *protocol.BlockPairContainer
	errs   bool
}{
	{&protocol.BlockPairContainer{}, true},
	{&protocol.BlockPairContainer{
		&protocol.TransactionsBlockContainer{},
		&protocol.ResultsBlockContainer{},
	}, true},
	{&protocol.BlockPairContainer{
		&protocol.TransactionsBlockContainer{
			Header:             (&protocol.TransactionsBlockHeaderBuilder{}).Build(),
			Metadata:           (&protocol.TransactionsBlockMetadataBuilder{}).Build(),
			SignedTransactions: []*protocol.SignedTransaction{},
			BlockProof:         (&protocol.TransactionsBlockProofBuilder{}).Build(),
		},
		&protocol.ResultsBlockContainer{
			Header:              (&protocol.ResultsBlockHeaderBuilder{}).Build(),
			TransactionReceipts: []*protocol.TransactionReceipt{},
			ContractStateDiffs:  []*protocol.ContractStateDiff{},
			BlockProof:          (&protocol.ResultsBlockProofBuilder{}).Build(),
		},
	}, false},
	{&protocol.BlockPairContainer{
		&protocol.TransactionsBlockContainer{
			Header: (&protocol.TransactionsBlockHeaderBuilder{
				NumSignedTransactions: 3,
			}).Build(),
			Metadata: (&protocol.TransactionsBlockMetadataBuilder{}).Build(),
			SignedTransactions: []*protocol.SignedTransaction{
				test.TransferTransaction().WithAmount(30).Build(),
				test.TransferTransaction().WithAmount(40).Build(),
				test.TransferTransaction().WithAmount(50).Build(),
			},
			BlockProof: (&protocol.TransactionsBlockProofBuilder{}).Build(),
		},
		&protocol.ResultsBlockContainer{
			Header: (&protocol.ResultsBlockHeaderBuilder{
				NumTransactionReceipts: 3,
				NumContractStateDiffs:  2,
			}).Build(),
			TransactionReceipts: []*protocol.TransactionReceipt{
				(&protocol.TransactionReceiptBuilder{}).Build(),
				(&protocol.TransactionReceiptBuilder{}).Build(),
				(&protocol.TransactionReceiptBuilder{}).Build(),
			},
			ContractStateDiffs: []*protocol.ContractStateDiff{
				(&protocol.ContractStateDiffBuilder{}).Build(),
				(&protocol.ContractStateDiffBuilder{}).Build(),
			},
			BlockProof: (&protocol.ResultsBlockProofBuilder{}).Build(),
		},
	}, false},
}

func TestBlockPair(t *testing.T) {
	for _, tt := range blockPairTable {
		payloads, err := encodeBlockPair(tt.origin)
		if tt.errs != (err != nil) {
			t.Fatalf("Expected error to be %v but got: %v", tt.errs, err)
		}
		if err != nil {
			continue
		}
		res, err := decodeBlockPair(payloads)
		if err != nil {
			t.Fatalf("Expected decode of passing encode not to return error but got: %v", err)
		}
		if !cmp.Equal(res, tt.origin) {
			t.Fatalf("Result and origin are different: %v", cmp.Diff(res, tt.origin))
		}
	}
}
