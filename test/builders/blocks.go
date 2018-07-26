package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto"
	"github.com/orbs-network/orbs-network-go/crypto/logic"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"time"
)

type blockPair struct {
	txHeader     *protocol.TransactionsBlockHeaderBuilder
	txMetadata   *protocol.TransactionsBlockMetadataBuilder
	transactions []*protocol.SignedTransaction
	txProof      *protocol.TransactionsBlockProofBuilder
	rxHeader     *protocol.ResultsBlockHeaderBuilder
	receipts     []*protocol.TransactionReceipt
	sdiffs       []*protocol.ContractStateDiff
	rxProof      *protocol.ResultsBlockProofBuilder
}

func BlockPair() *blockPair {
	createdDate := time.Now()
	return &blockPair{
		txHeader: &protocol.TransactionsBlockHeaderBuilder{
			BlockHeight:           1,
			Timestamp:             primitives.TimestampNano(createdDate.UnixNano()),
			ProtocolVersion:       primitives.ProtocolVersion(blockstorage.ProtocolVersion),
			NumSignedTransactions: 1,
		},
		txMetadata: &protocol.TransactionsBlockMetadataBuilder{},
		transactions: []*protocol.SignedTransaction{
			(TransferTransaction().WithAmount(10)).Build(),
		},
		txProof: &protocol.TransactionsBlockProofBuilder{
			Type: protocol.TRANSACTIONS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		},
		rxHeader: &protocol.ResultsBlockHeaderBuilder{
			BlockHeight:            1,
			Timestamp:              primitives.TimestampNano(createdDate.UnixNano()),
			ProtocolVersion:        primitives.ProtocolVersion(blockstorage.ProtocolVersion),
			NumContractStateDiffs:  1,
			NumTransactionReceipts: 1,
		},
		receipts: []*protocol.TransactionReceipt{
			(TransactionReceipt().Build()),
		},
		sdiffs: []*protocol.ContractStateDiff{
			(ContractStateDiff().Build()),
		},
		rxProof: &protocol.ResultsBlockProofBuilder{
			Type: protocol.RESULTS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		},
	}
}

func LeanHelixBlockPair() *blockPair {
	return BlockPair()
}

func BenchmarkConsensusBlockPair() *blockPair {
	return BlockPair().WithBenchmarkConsensusBlockProof(nil, []byte{0x88})
}

func (b *blockPair) Build() *protocol.BlockPairContainer {
	return &protocol.BlockPairContainer{
		TransactionsBlock: &protocol.TransactionsBlockContainer{
			Header:             b.txHeader.Build(),
			Metadata:           b.txMetadata.Build(),
			SignedTransactions: b.transactions,
			BlockProof:         b.txProof.Build(),
		},
		ResultsBlock: &protocol.ResultsBlockContainer{
			Header:              b.rxHeader.Build(),
			TransactionReceipts: b.receipts,
			ContractStateDiffs:  b.sdiffs,
			BlockProof:          b.rxProof.Build(),
		},
	}
}

func (b *blockPair) WithHeight(blockHeight primitives.BlockHeight) *blockPair {
	b.txHeader.BlockHeight = blockHeight
	b.rxHeader.BlockHeight = blockHeight
	return b
}

func (b *blockPair) WithPrevBlockHash(prevBlock *protocol.BlockPairContainer) *blockPair {
	b.txHeader.PrevBlockHashPtr = crypto.CalcTransactionsBlockHash(prevBlock)
	b.rxHeader.PrevBlockHashPtr = crypto.CalcResultsBlockHash(prevBlock)
	return b
}

func (b *blockPair) WithBenchmarkConsensusBlockProof(privateKey primitives.Ed25519PrivateKey, publicKey primitives.Ed25519PublicKey) *blockPair {
	built := b.Build()
	txHash := crypto.CalcTransactionsBlockHash(built)
	rxHash := crypto.CalcResultsBlockHash(built)
	xorHash := logic.CalcXor(txHash, rxHash)
	sig := signature.SignEd25519(privateKey, xorHash)
	b.rxProof = &protocol.ResultsBlockProofBuilder{
		Type: protocol.RESULTS_BLOCK_PROOF_TYPE_BENCHMARK_CONSENSUS,
		BenchmarkConsensus: &consensus.BenchmarkConsensusBlockProofBuilder{
			Sender: &consensus.BenchmarkConsensusSenderSignatureBuilder{
				SenderPublicKey: publicKey,
				Signature:       sig,
			},
		},
	}
	return b
}

func (b *blockPair) WithInvalidBenchmarkConsensusBlockProof(privateKey primitives.Ed25519PrivateKey, publicKey primitives.Ed25519PublicKey) *blockPair {
	built := b.Build()
	txHash := crypto.CalcTransactionsBlockHash(built)
	rxHash := crypto.CalcResultsBlockHash(built)
	xorHash := logic.CalcXor(txHash, rxHash)
	signature := signature.SignEd25519(privateKey, xorHash)
	signature[0] ^= 0xaa // corrupt the first byte of the signature
	b.rxProof = &protocol.ResultsBlockProofBuilder{
		Type: protocol.RESULTS_BLOCK_PROOF_TYPE_BENCHMARK_CONSENSUS,
		BenchmarkConsensus: &consensus.BenchmarkConsensusBlockProofBuilder{
			Sender: &consensus.BenchmarkConsensusSenderSignatureBuilder{
				SenderPublicKey: publicKey,
				Signature:       signature,
			},
		},
	}
	return b
}

func (b *blockPair) WithBlockCreated(time time.Time) *blockPair {
	b.txHeader.Timestamp = primitives.TimestampNano(time.UnixNano())
	b.rxHeader.Timestamp = primitives.TimestampNano(time.UnixNano())
	return b
}

func (b *blockPair) WithProtocolVersion(version primitives.ProtocolVersion) *blockPair {
	b.txHeader.ProtocolVersion = version
	b.rxHeader.ProtocolVersion = version
	return b
}

func (b *blockPair) WithTransactions(num uint32) *blockPair {
	b.transactions = make([]*protocol.SignedTransaction, 0, num)
	for i := uint32(0); i < num; i++ {
		b.transactions = append(b.transactions, TransferTransaction().WithAmount(uint64(10*num)).Build())
	}
	b.txHeader.NumSignedTransactions = num
	return b
}

func (b *blockPair) WithReceipts(num uint32) *blockPair {
	b.receipts = make([]*protocol.TransactionReceipt, 0, num)
	for i := uint32(0); i < num; i++ {
		b.receipts = append(b.receipts, TransactionReceipt().Build())
	}
	b.rxHeader.NumTransactionReceipts = num
	return b
}

func (b *blockPair) WithStateDiffs(num uint32) *blockPair {
	b.sdiffs = make([]*protocol.ContractStateDiff, 0, num)
	for i := uint32(0); i < num; i++ {
		b.sdiffs = append(b.sdiffs, ContractStateDiff().Build())
	}
	b.rxHeader.NumContractStateDiffs = num
	return b
}

func (b *blockPair) WithCorruptNumTransactions(num uint32) *blockPair {
	b.transactions = []*protocol.SignedTransaction{}
	b.txHeader.NumSignedTransactions = num
	return b
}

func (b *blockPair) WithCorruptNumReceipts(num uint32) *blockPair {
	b.receipts = []*protocol.TransactionReceipt{}
	b.rxHeader.NumTransactionReceipts = num
	return b
}

func (b *blockPair) WithCorruptNumStateDiffs(num uint32) *blockPair {
	b.sdiffs = []*protocol.ContractStateDiff{}
	b.rxHeader.NumContractStateDiffs = num
	return b
}

type corruptBlockPair struct {
	txContainer *protocol.TransactionsBlockContainer
	rxContainer *protocol.ResultsBlockContainer
}

func CorruptBlockPair() *corruptBlockPair {
	return &corruptBlockPair{}
}

func (c *corruptBlockPair) Build() *protocol.BlockPairContainer {
	return &protocol.BlockPairContainer{
		TransactionsBlock: c.txContainer,
		ResultsBlock:      c.rxContainer,
	}
}

func (c *corruptBlockPair) WithMissingTransactionsBlock() *corruptBlockPair {
	c.txContainer = nil
	return c
}

func (c *corruptBlockPair) WithMissingResultsBlock() *corruptBlockPair {
	c.rxContainer = nil
	return c
}

func (c *corruptBlockPair) WithEmptyTransactionsBlock() *corruptBlockPair {
	c.txContainer = &protocol.TransactionsBlockContainer{}
	return c
}

func (c *corruptBlockPair) WithEmptyResultsBlock() *corruptBlockPair {
	c.rxContainer = &protocol.ResultsBlockContainer{}
	return c
}
