package test

import (
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"time"
)

type blockPairBuilder struct {
	height          int
	createdDate     time.Time
	transactions    []*protocol.SignedTransaction
	protocolVersion int
}

func BlockPairBuilder() *blockPairBuilder {
	return &blockPairBuilder{
		height:          1,
		createdDate:     time.Now(),
		protocolVersion: blockstorage.ProtocolVersion,
		transactions: []*protocol.SignedTransaction{
			(TransferTransaction().WithAmount(10)).Build(),
		},
	}
}

func (b *blockPairBuilder) Build() *protocol.BlockPairContainer {

	height := primitives.BlockHeight(b.height)
	blockCreated := primitives.TimestampNano(b.createdDate.UnixNano())

	transactionsBlock := &protocol.TransactionsBlockContainer{
		Header: (&protocol.TransactionsBlockHeaderBuilder{
			BlockHeight:     height,
			Timestamp:       blockCreated,
			ProtocolVersion: primitives.ProtocolVersion(b.protocolVersion),
		}).Build(),
		BlockProof: (&protocol.TransactionsBlockProofBuilder{
			Type: protocol.TRANSACTIONS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		}).Build(),
		Metadata:           (&protocol.TransactionsBlockMetadataBuilder{}).Build(),
		SignedTransactions: b.transactions,
	}

	resultsBlock := &protocol.ResultsBlockContainer{
		Header: (&protocol.ResultsBlockHeaderBuilder{
			BlockHeight:            height,
			Timestamp:              blockCreated,
			NumContractStateDiffs:  1,
			NumTransactionReceipts: 1,
		}).Build(),
		BlockProof: (&protocol.ResultsBlockProofBuilder{
			Type: protocol.RESULTS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		}).Build(),
		ContractStateDiffs: []*protocol.ContractStateDiff{
			(&protocol.ContractStateDiffBuilder{
				ContractName: "BenchmarkToken",
				StateDiffs: []*protocol.StateRecordBuilder{
					{Key: []byte("amount"), Value: []byte{10}},
				},
			}).Build(),
		},
		TransactionReceipts: []*protocol.TransactionReceipt{
			(&protocol.TransactionReceiptBuilder{
				Txhash:          []byte("some-tx-hash"),
				ExecutionResult: protocol.EXECUTION_RESULT_SUCCESS,
			}).Build(),
		},
	}

	container := &protocol.BlockPairContainer{
		TransactionsBlock: transactionsBlock,
		ResultsBlock:      resultsBlock,
	}

	return container
}

func (b *blockPairBuilder) WithHeight(height int) *blockPairBuilder {
	b.height = height
	return b
}

func (b *blockPairBuilder) WithBlockCreated(time time.Time) *blockPairBuilder {
	b.createdDate = time
	return b
}

func (b *blockPairBuilder) WithProtocolVersion(version int) *blockPairBuilder {
	b.protocolVersion = version
	return b
}
