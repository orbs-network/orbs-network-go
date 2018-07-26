package crypto

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

func CalcTransactionsBlockHash(blockPair *protocol.BlockPairContainer) primitives.Sha256 {
	return hash.CalcSha256(blockPair.TransactionsBlock.Header.Raw())
}

func CalcResultsBlockHash(blockPair *protocol.BlockPairContainer) primitives.Sha256 {
	return hash.CalcSha256(blockPair.ResultsBlock.Header.Raw())
}
