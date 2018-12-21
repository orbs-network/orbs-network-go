package keys

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGenerateEd25519Key(t *testing.T) {
	keyPair, err := GenerateEd25519Key()
	require.NoError(t, err, "should not fail")

	t.Logf("Public: %s", keyPair.PublicKeyHex())
	t.Logf("Private: %s", keyPair.PrivateKeyHex())
	require.Equal(t, ED25519_PUBLIC_KEY_SIZE_BYTES, len(keyPair.publicKey), "public key length should match")
	require.Equal(t, ED25519_PRIVATE_KEY_SIZE_BYTES, len(keyPair.privateKey), "private key length should match")
}
