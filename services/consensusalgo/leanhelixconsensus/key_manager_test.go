package leanhelixconsensus

import (
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSignAndVerifyConsensusMessage(t *testing.T) {

	keyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)
	mgr := NewKeyManager(keyPair.PrivateKey())
	content := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	contentSig := mgr.SignConsensusMessage(1, content)
	verified := mgr.VerifyConsensusMessage(1, content, contentSig, lhprimitives.MemberId(keyPair.NodeAddress()))
	require.True(t, verified, "Verification of original consensus message should succeed")
}

func TestSignAndVerifyConsensusMessageOfMismatchedHeight(t *testing.T) {
	t.Skip("Remove the skip when block height is actually verified by VerifyConsensusMessage()")
	keyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)
	mgr := NewKeyManager(keyPair.PrivateKey())
	content := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	contentSig := mgr.SignConsensusMessage(1, content)
	verified := mgr.VerifyConsensusMessage(2, content, contentSig, lhprimitives.MemberId(keyPair.NodeAddress()))
	require.False(t, verified, "Verification of consensus message that was signed for another block height should fail")
}

func TestSignAndVerifyTaintedConsensusMessage(t *testing.T) {

	keyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)
	mgr := NewKeyManager(keyPair.PrivateKey())
	content := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	tamperedMessage := []byte{0, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	contentSig := mgr.SignConsensusMessage(1, content)
	verified := mgr.VerifyConsensusMessage(1, tamperedMessage, contentSig, lhprimitives.MemberId(keyPair.NodeAddress()))
	require.False(t, verified, "Verification of a tampered consensus message should fail")
}

func TestSignAndVerifyRandomSeed(t *testing.T) {

	keyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)
	mgr := NewKeyManager(keyPair.PrivateKey())
	randomSeed := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	randomSeedSig := mgr.SignRandomSeed(1, randomSeed)
	verified := mgr.VerifyRandomSeed(1, randomSeed, randomSeedSig, lhprimitives.MemberId(keyPair.NodeAddress()))
	require.True(t, verified, "Verification of original random seed should succeed")
}

func TestSignAndVerifyTaintedRandomSeed(t *testing.T) {

	keyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)
	mgr := NewKeyManager(keyPair.PrivateKey())
	randomSeed := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	tamperedRandomSeed := []byte{0, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	randomSeedSig := mgr.SignRandomSeed(1, randomSeed)
	verified := mgr.VerifyRandomSeed(1, tamperedRandomSeed, randomSeedSig, lhprimitives.MemberId(keyPair.NodeAddress()))
	require.False(t, verified, "Verification of a tampered random seed should fail")
}

func TestSignAndVerifyRandomSeedOfMismatchedHeight(t *testing.T) {
	t.Skip("Remove the skip when block height is actually verified by VerifyRandomSeed()")
	keyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)
	mgr := NewKeyManager(keyPair.PrivateKey())
	randomSeed := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	randomSeedSig := mgr.SignRandomSeed(1, randomSeed)
	verified := mgr.VerifyRandomSeed(2, randomSeed, randomSeedSig, lhprimitives.MemberId(keyPair.NodeAddress()))
	require.False(t, verified, "Verification of random seed that was signed for another block height should fail")

}
