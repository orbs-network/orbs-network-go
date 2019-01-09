package digest

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

func CalcQueryHash(query *protocol.Query) primitives.Sha256 {
	return hash.CalcSha256(query.Raw())
}
