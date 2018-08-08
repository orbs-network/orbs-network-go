package digest

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

func CalcTransactionsBlockHash(transactionsBlock *protocol.TransactionsBlockContainer) primitives.Sha256 {
	return hash.CalcSha256(transactionsBlock.Header.Raw())
}

func CalcResultsBlockHash(resultsBlock *protocol.ResultsBlockContainer) primitives.Sha256 {
	return hash.CalcSha256(resultsBlock.Header.Raw())
}
