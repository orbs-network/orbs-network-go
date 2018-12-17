package digest

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	ExamplePublicKey    = "30fccea741dd34c7afb146a543616bcb361247148f0c8542541c01da6d6cadf186515f1d851978fc94a6a641e25dec74a6ec28c5ae04c651a0dc2e6104b3ac24"
	ExamplePrivateKey   = "901a1a0bfbe217593062a054e561e708707cb814a123474c25fd567a0fe088f8"
	ExpectedNodeAddress = "a328846cd5b4979d68a8c58a9bdfeee657b34de7"
)

func TestCalcNodeAddressFromPublicKey(t *testing.T) {
	publicKey, _ := hex.DecodeString(ExamplePublicKey)
	nodeAddress := CalcNodeAddressFromPublicKey(primitives.EcdsaSecp256K1PublicKey(publicKey))
	require.Equal(t, ExpectedNodeAddress, nodeAddress.String(), "result should match")
}

func TestVerifyEcdsaSecp256K1WithNodeAddress(t *testing.T) {
	dataToSign := hash.CalcSha256([]byte("this is what we want to sign"))
	privateKey, _ := hex.DecodeString(ExamplePrivateKey)
	nodeAddress, _ := hex.DecodeString(ExpectedNodeAddress)

	sig, err := signature.SignEcdsaSecp256K1(privateKey, dataToSign)
	require.NoError(t, err)

	ok := VerifyEcdsaSecp256K1WithNodeAddress(nodeAddress, dataToSign, sig)
	require.True(t, ok, "verification should succeed")
}
