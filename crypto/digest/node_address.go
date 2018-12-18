package digest

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

const (
	NODE_ADDRESS_SIZE_BYTES = 20
)

func CalcNodeAddressFromPublicKey(publicKey primitives.EcdsaSecp256K1PublicKey) primitives.NodeAddress {
	return primitives.NodeAddress(hash.CalcKeccak256(publicKey)[12:])
}
