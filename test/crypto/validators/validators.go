// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package validators

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

// TODO: this file should be moved to /test/builders/blocks.go

func AStructurallyValidBlock() *protocol.BlockPairContainer {

	protocolVersion := primitives.ProtocolVersion(1)
	virtualChainId := primitives.VirtualChainId(1)
	currentBlockHeight := primitives.BlockHeight(1000)
	validPrevBlock := builders.BlockPair().WithHeight(currentBlockHeight - 1).Build()
	validPrevBlockHash := digest.CalcTransactionsBlockHash(validPrevBlock.TransactionsBlock)
	//validPrevBlockTimestamp := primitives.TimestampNano(time.Now().UnixNano() - 1000)
	transaction := builders.TransferTransaction().WithAmountAndTargetAddress(10, builders.ClientAddressForEd25519SignerForTests(6)).Build()
	txMetadata := &protocol.TransactionsBlockMetadataBuilder{}
	txRootHashForValidBlock, _ := digest.CalcTransactionsMerkleRoot([]*protocol.SignedTransaction{transaction})
	validMetadataHash := digest.CalcTransactionMetaDataHash(txMetadata.Build())

	block := builders.
		BlockPair().
		WithHeight(currentBlockHeight).
		WithProtocolVersion(protocolVersion).
		WithVirtualChainId(virtualChainId).
		WithTransactions(0).
		WithTransaction(transaction).
		WithPrevBlock(validPrevBlock).
		WithPrevBlockHash(validPrevBlockHash).
		WithMetadata(txMetadata).
		WithMetadataHash(validMetadataHash).
		WithTransactionsRootHash(txRootHashForValidBlock).
		Build()

	txBlockHashPtr := digest.CalcTransactionsBlockHash(block.TransactionsBlock)
	receiptMerkleRoot, _ := digest.CalcReceiptsMerkleRoot(block.ResultsBlock.TransactionReceipts)
	stateDiffHash, _ := digest.CalcStateDiffHash(block.ResultsBlock.ContractStateDiffs)
	preExecutionRootHash := &services.GetStateHashOutput{
		StateMerkleRootHash: hash.CalcSha256([]byte{1, 2, 3, 4, 3, 2, 1}),
	}

	block.ResultsBlock.Header.MutateTransactionsBlockHashPtr(txBlockHashPtr)
	block.ResultsBlock.Header.MutateReceiptsMerkleRootHash(receiptMerkleRoot)
	block.ResultsBlock.Header.MutateStateDiffHash(stateDiffHash)
	block.ResultsBlock.Header.MutatePreExecutionStateMerkleRootHash(preExecutionRootHash.StateMerkleRootHash)

	return block
}

func MockCalcReceiptsMerkleRootThatReturns(root primitives.Sha256, err error) func(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error) {
	return func(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error) {
		return root, err
	}
}

func MockCalcStateDiffHashThatReturns(root primitives.Sha256, err error) func(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error) {
	return func(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error) {
		return root, err
	}
}
