package statestorage

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMerkleEntryGenerator(t *testing.T) {

	k1, v1 := getMerkleEntry("c", primitives.Ripmd160Sha256("k"), []byte("v"))
	k2, v2 := getMerkleEntry("c", primitives.Ripmd160Sha256("k1"), []byte("v"))
	k3, v3 := getMerkleEntry("c1", primitives.Ripmd160Sha256("k"), []byte("v"))

	require.Len(t, k1, 32)
	require.Len(t, k2, 32)
	require.Len(t, k3, 32)
	require.Len(t, v1, 32)
	require.Len(t, v2, 32)
	require.Len(t, v3, 32)

	require.Equal(t, v1, v2)
	require.Equal(t, v1, v3)

	require.NotEqual(t, k1, k2)
	require.NotEqual(t, k1, k3)
	require.NotEqual(t, k2, k3)

	k1, v1 = getMerkleEntry("c", primitives.Ripmd160Sha256("k"), []byte("v1"))
	k2, v2 = getMerkleEntry("c", primitives.Ripmd160Sha256("k"), []byte("v2"))

	require.Len(t, k1, 32)
	require.Len(t, k2, 32)
	require.Len(t, v1, 32)
	require.Len(t, v2, 32)

	require.NotEqual(t, v1, v2)

	require.Equal(t, k1, k2)
}
