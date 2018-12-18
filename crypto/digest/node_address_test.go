package digest

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	ExampleNodePublicKey = "30fccea741dd34c7afb146a543616bcb361247148f0c8542541c01da6d6cadf186515f1d851978fc94a6a641e25dec74a6ec28c5ae04c651a0dc2e6104b3ac24"
	ExpectedNodeAddress  = "a328846cd5b4979d68a8c58a9bdfeee657b34de7"
)

func TestCalcNodeAddressFromPublicKey(t *testing.T) {
	publicKey, _ := hex.DecodeString(ExampleNodePublicKey)
	nodeAddress := CalcNodeAddressFromPublicKey(primitives.EcdsaSecp256K1PublicKey(publicKey))
	require.Equal(t, ExpectedNodeAddress, nodeAddress.String(), "result should match")
}
