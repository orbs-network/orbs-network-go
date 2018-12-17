package digest

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

const (
	NODE_ADDRESS_SIZE_BYTES = 20
)

func CalcNodeAddressFromPublicKey(publicKey primitives.EcdsaSecp256K1PublicKey) primitives.NodeAddress {
	return primitives.NodeAddress(hash.CalcKeccak256(publicKey)[12:])
}

func VerifyEcdsaSecp256K1WithNodeAddress(nodeAddress primitives.NodeAddress, data []byte, sig primitives.EcdsaSecp256K1Sig) bool {
	if len(nodeAddress) != NODE_ADDRESS_SIZE_BYTES {
		return false
	}
	publicKey, err := signature.RecoverEcdsaSecp256K1(data, sig)
	if err != nil {
		return false
	}
	recoveredNodeAddress := CalcNodeAddressFromPublicKey(publicKey)
	return nodeAddress.Equal(recoveredNodeAddress)
}
