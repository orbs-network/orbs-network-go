// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package codec

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

const NUM_HARDCODED_PAYLOADS_FOR_BLOCK_PAIR = 5 // txHeader, txMetadata, rxHeader..

func EncodeBlockPair(blockPair *protocol.BlockPairContainer) ([][]byte, error) {
	if blockPair == nil || blockPair.TransactionsBlock == nil || blockPair.ResultsBlock == nil {
		return nil, errors.Errorf("codec failed to encode block pair due to missing fields: %s", blockPair.String())
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
		return nil, errors.Errorf("codec failed to encode block pair due to missing fields: %s", blockPair.String())
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

func EncodeBlockPairs(blockPairs []*protocol.BlockPairContainer) ([][]byte, error) {
	var payloads [][]byte

	for _, blocks := range blockPairs {
		blockPairPayloads, err := EncodeBlockPair(blocks)
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, blockPairPayloads...)
	}

	return payloads, nil
}

func DecodeBlockPair(payloads [][]byte) (*protocol.BlockPairContainer, error) {
	results, err := DecodeBlockPairs(payloads)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, errors.New("codec failed to decode at least one block pair")
	}

	return results[0], nil
}

func DecodeBlockPairs(payloads [][]byte) (results []*protocol.BlockPairContainer, err error) {
	payloadIndex := uint32(0)

	for payloadIndex < uint32(len(payloads)) {
		if uint32(len(payloads)) < payloadIndex+NUM_HARDCODED_PAYLOADS_FOR_BLOCK_PAIR {
			return nil, errors.Errorf("codec failed to decode block pair, missing payloads %d", len(payloads))
		}

		txBlockHeader := protocol.TransactionsBlockHeaderReader(payloads[payloadIndex])
		txBlockMetadata := protocol.TransactionsBlockMetadataReader(payloads[payloadIndex+1])
		txBlockProof := protocol.TransactionsBlockProofReader(payloads[payloadIndex+2])
		rxBlockHeader := protocol.ResultsBlockHeaderReader(payloads[payloadIndex+3])
		rxBlockProof := protocol.ResultsBlockProofReader(payloads[payloadIndex+4])
		payloadIndex += uint32(NUM_HARDCODED_PAYLOADS_FOR_BLOCK_PAIR)

		expectedPayloads := txBlockHeader.NumSignedTransactions() + rxBlockHeader.NumTransactionReceipts() + rxBlockHeader.NumContractStateDiffs()
		if uint32(len(payloads)) < payloadIndex+expectedPayloads {
			return nil, errors.Errorf("codec failed to decode block pair, remaining payloads %d, expected payloads %d", uint32(len(payloads))-payloadIndex, expectedPayloads)
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

		payloadIndex += rxBlockHeader.NumContractStateDiffs()

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
		results = append(results, blockPair)
	}
	return results, nil
}
