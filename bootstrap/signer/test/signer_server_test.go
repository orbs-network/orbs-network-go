package test

import (
	"github.com/orbs-network/orbs-network-go/bootstrap/signer"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	crypto "github.com/orbs-network/orbs-network-go/crypto/signer"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type signerServerConfig struct {
	privateKey primitives.EcdsaSecp256K1PrivateKey
	address    string
}

func (s *signerServerConfig) NodePrivateKey() primitives.EcdsaSecp256K1PrivateKey {
	return s.privateKey
}

func (s *signerServerConfig) HttpAddress() string {
	return s.address
}

func TestSignerClient(t *testing.T) {
	address := "localhost:9999"
	pk := keys.EcdsaSecp256K1KeyPairForTests(0).PrivateKey()

	testOutput := log.NewTestOutput(t, log.NewHumanReadableFormatter())
	testLogger := log.GetLogger().WithOutput(testOutput)

	server := signer.StartSignerServer(&signerServerConfig{pk, address}, testLogger)
	defer server.GracefulShutdown(1 * time.Second)
	c := crypto.NewSignerClient("http://" + address)

	payload := []byte("payload")

	signed, err := digest.SignAsNode(pk, payload)
	require.NoError(t, err)

	clientSignature, err := c.Sign(payload)
	require.NoError(t, err)

	require.EqualValues(t, signed, clientSignature)
}
