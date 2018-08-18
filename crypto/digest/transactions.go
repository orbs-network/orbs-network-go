package digest

import (
	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

func CalcTxHash(transaction *protocol.Transaction) primitives.Sha256 {
	return hash.CalcSha256(transaction.Raw())
}

func CalcTxId(transaction *protocol.Transaction) []byte {
	return GenerateTxId(CalcTxHash(transaction), transaction.Timestamp())
}

func GenerateTxId(hash primitives.Sha256, ts primitives.TimestampNano) []byte {
	result := make([]byte, 8+32)
	membuffers.WriteUint64(result, uint64(ts))
	copy(result[8:], hash)

	return result
}
