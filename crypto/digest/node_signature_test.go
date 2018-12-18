package digest

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	ExampleNodePrivateKey = "901a1a0bfbe217593062a054e561e708707cb814a123474c25fd567a0fe088f8"
	DifferentNodeAddress  = "bb51846fd3b4110d6818c55a9bdfe9e657b33300"
)

var ExampleDataToSign = []byte("this is what we want to sign")

func TestVerifyNodeSignature(t *testing.T) {
	privateKey, _ := hex.DecodeString(ExampleNodePrivateKey)
	nodeAddress, _ := hex.DecodeString(ExpectedNodeAddress)

	sig, err := SignAsNode(privateKey, ExampleDataToSign)
	require.NoError(t, err)

	ok := VerifyNodeSignature(nodeAddress, ExampleDataToSign, sig)
	require.True(t, ok, "verification should succeed")
}

func TestVerifyNodeSignature_InvalidAddress(t *testing.T) {
	privateKey, _ := hex.DecodeString(ExampleNodePrivateKey)
	differentNodeAddress, _ := hex.DecodeString(DifferentNodeAddress)

	sig, err := SignAsNode(privateKey, ExampleDataToSign)
	require.NoError(t, err)

	ok := VerifyNodeSignature(differentNodeAddress, ExampleDataToSign, sig)
	require.False(t, ok, "verification should fail")
}
