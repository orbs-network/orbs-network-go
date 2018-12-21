package digest

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

// don't need to provide hashed data as this function will SHA256
func SignAsNode(privateKey primitives.EcdsaSecp256K1PrivateKey, data []byte) (primitives.EcdsaSecp256K1Sig, error) {
	hashedData := hash.CalcSha256(data)
	return signature.SignEcdsaSecp256K1(privateKey, hashedData)
}

func VerifyNodeSignature(nodeAddress primitives.NodeAddress, data []byte, sig primitives.EcdsaSecp256K1Sig) bool {
	if len(nodeAddress) != NODE_ADDRESS_SIZE_BYTES {
		return false
	}
	hashedData := hash.CalcSha256(data)
	publicKey, err := signature.RecoverEcdsaSecp256K1(hashedData, sig)
	if err != nil {
		return false
	}
	recoveredNodeAddress := CalcNodeAddressFromPublicKey(publicKey)
	return nodeAddress.Equal(recoveredNodeAddress)
}
