package validators

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

func BuildValidTestBlock() *protocol.BlockPairContainer {

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

	return block
}
