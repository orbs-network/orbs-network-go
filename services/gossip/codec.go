package gossip

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

const NUM_HARDCODED_PAYLOADS_FOR_BLOCK_PAIR = 5 // txHeader, txMetadata, rxHeader..

func encodeBlockPair(blockPair *protocol.BlockPairContainer) ([][]byte, error) {
	if blockPair == nil || blockPair.TransactionsBlock == nil || blockPair.ResultsBlock == nil {
		return nil, errors.Errorf("codec failed to encode block pair due to missing fields", log.Stringable("block-pair", blockPair))
	}

	payloads := make([][]byte, 0, NUM_HARDCODED_PAYLOADS_FOR_BLOCK_PAIR+
		len(blockPair.TransactionsBlock.SignedTransactions)+
		len(blockPair.ResultsBlock.TransactionReceipts)+
		len(blockPair.ResultsBlock.ContractStateDiffs),
	)

	if blockPair.TransactionsBlock.Header == nil ||
		blockPair.TransactionsBlock.Metadata == nil ||
		blockPair.TransactionsBlock.BlockProof == nil ||
		blockPair.ResultsBlock.Header == nil ||
		blockPair.ResultsBlock.BlockProof == nil {
		return nil, errors.Errorf("codec failed to encode block pair due to missing fields", log.Stringable("block-pair", blockPair))
	}

	payloads = append(payloads, blockPair.TransactionsBlock.Header.Raw())
	payloads = append(payloads, blockPair.TransactionsBlock.Metadata.Raw())
	payloads = append(payloads, blockPair.TransactionsBlock.BlockProof.Raw())
	payloads = append(payloads, blockPair.ResultsBlock.Header.Raw())
	payloads = append(payloads, blockPair.ResultsBlock.BlockProof.Raw())

	for _, tx := range blockPair.TransactionsBlock.SignedTransactions {
		payloads = append(payloads, tx.Raw())
	}

	for _, receipt := range blockPair.ResultsBlock.TransactionReceipts {
		payloads = append(payloads, receipt.Raw())
	}

	for _, sdiff := range blockPair.ResultsBlock.ContractStateDiffs {
		payloads = append(payloads, sdiff.Raw())
	}
	return payloads, nil
}

func decodeBlockPair(payloads [][]byte) (*protocol.BlockPairContainer, error) {
	if len(payloads) < NUM_HARDCODED_PAYLOADS_FOR_BLOCK_PAIR {
		return nil, errors.Errorf("codec failed to decode block pair due to missing payloads", log.Int("num-payloads", len(payloads)))
	}

	txBlockHeader := protocol.TransactionsBlockHeaderReader(payloads[0])
	txBlockMetadata := protocol.TransactionsBlockMetadataReader(payloads[1])
	txBlockProof := protocol.TransactionsBlockProofReader(payloads[2])
	rxBlockHeader := protocol.ResultsBlockHeaderReader(payloads[3])
	rxBlockProof := protocol.ResultsBlockProofReader(payloads[4])
	payloadIndex := uint32(NUM_HARDCODED_PAYLOADS_FOR_BLOCK_PAIR)

	expectedPayloads := NUM_HARDCODED_PAYLOADS_FOR_BLOCK_PAIR + txBlockHeader.NumSignedTransactions() + rxBlockHeader.NumTransactionReceipts() + rxBlockHeader.NumContractStateDiffs()
	if uint32(len(payloads)) < expectedPayloads {
		return nil, errors.Errorf("codec failed to decode block pair due to missing payloads", log.Int("num-payloads", len(payloads)), log.Uint32("expected-payloads", expectedPayloads))
	}

	txs := make([]*protocol.SignedTransaction, 0, txBlockHeader.NumSignedTransactions())
	for i := uint32(0); i < txBlockHeader.NumSignedTransactions(); i++ {
		txs = append(txs, protocol.SignedTransactionReader(payloads[payloadIndex+i]))
	}
	payloadIndex += txBlockHeader.NumSignedTransactions()

	receipts := make([]*protocol.TransactionReceipt, 0, rxBlockHeader.NumTransactionReceipts())
	for i := uint32(0); i < rxBlockHeader.NumTransactionReceipts(); i++ {
		receipts = append(receipts, protocol.TransactionReceiptReader(payloads[payloadIndex+i]))
	}
	payloadIndex += rxBlockHeader.NumTransactionReceipts()

	sdiffs := make([]*protocol.ContractStateDiff, 0, rxBlockHeader.NumContractStateDiffs())
	for i := uint32(0); i < rxBlockHeader.NumContractStateDiffs(); i++ {
		sdiffs = append(sdiffs, protocol.ContractStateDiffReader(payloads[payloadIndex+i]))
	}

	blockPair := &protocol.BlockPairContainer{
		TransactionsBlock: &protocol.TransactionsBlockContainer{
			Header:             txBlockHeader,
			Metadata:           txBlockMetadata,
			SignedTransactions: txs,
			BlockProof:         txBlockProof,
		},
		ResultsBlock: &protocol.ResultsBlockContainer{
			Header:              rxBlockHeader,
			TransactionReceipts: receipts,
			ContractStateDiffs:  sdiffs,
			BlockProof:          rxBlockProof,
		},
	}
	return blockPair, nil
}
