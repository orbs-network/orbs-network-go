// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package leanhelixconsensus

import (
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	lhprotocol "github.com/orbs-network/lean-helix-go/spec/types/go/protocol"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSignAndVerifyConsensusMessage(t *testing.T) {

	keyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)
	mgr := NewKeyManager(log.DefaultTestingLogger(t), keyPair.PrivateKey())
	content := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	contentSig := mgr.SignConsensusMessage(1, content)
	senderSignature := lhprotocol.SenderSignatureBuilder{
		MemberId:  lhprimitives.MemberId(keyPair.NodeAddress()),
		Signature: contentSig,
	}
	err := mgr.VerifyConsensusMessage(1, content, senderSignature.Build())
	require.NoError(t, err, "Verification of original consensus message should succeed")
}

func TestSignAndVerifyConsensusMessageOfMismatchedHeight(t *testing.T) {
	t.Skip("Remove the skip when block height is actually verified by VerifyConsensusMessage()")
	keyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)
	mgr := NewKeyManager(log.DefaultTestingLogger(t), keyPair.PrivateKey())
	content := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	contentSig := mgr.SignConsensusMessage(1, content)
	senderSignature := lhprotocol.SenderSignatureBuilder{
		MemberId:  lhprimitives.MemberId(keyPair.NodeAddress()),
		Signature: contentSig,
	}

	err := mgr.VerifyConsensusMessage(2, content, senderSignature.Build())
	require.Error(t, err, "Verification of consensus message that was signed for another block height should fail")
}

func TestSignAndVerifyTaintedConsensusMessage(t *testing.T) {

	keyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)
	mgr := NewKeyManager(log.DefaultTestingLogger(t), keyPair.PrivateKey())
	content := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	tamperedMessage := []byte{0, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	contentSig := mgr.SignConsensusMessage(1, content)
	senderSignature := lhprotocol.SenderSignatureBuilder{
		MemberId:  lhprimitives.MemberId(keyPair.NodeAddress()),
		Signature: contentSig,
	}
	err := mgr.VerifyConsensusMessage(1, tamperedMessage, senderSignature.Build())
	require.Error(t, err, "Verification of a tampered consensus message should fail")
}

func TestSignAndVerifyRandomSeed(t *testing.T) {

	keyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)
	mgr := NewKeyManager(log.DefaultTestingLogger(t), keyPair.PrivateKey())
	randomSeed := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	randomSeedSig := mgr.SignRandomSeed(1, randomSeed)
	senderSignature := lhprotocol.SenderSignatureBuilder{
		MemberId:  lhprimitives.MemberId(keyPair.NodeAddress()),
		Signature: lhprimitives.Signature(randomSeedSig),
	}
	err := mgr.VerifyRandomSeed(1, randomSeed, senderSignature.Build())
	require.Nil(t, err, "Verification of original random seed should succeed")
}

func TestSignAndVerifyTaintedRandomSeed(t *testing.T) {

	keyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)
	mgr := NewKeyManager(log.DefaultTestingLogger(t), keyPair.PrivateKey())
	randomSeed := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	tamperedRandomSeed := []byte{0, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	randomSeedSig := mgr.SignRandomSeed(1, randomSeed)
	senderSignature := lhprotocol.SenderSignatureBuilder{
		MemberId:  lhprimitives.MemberId(keyPair.NodeAddress()),
		Signature: lhprimitives.Signature(randomSeedSig),
	}
	err := mgr.VerifyRandomSeed(1, tamperedRandomSeed, senderSignature.Build())
	require.Error(t, err, "Verification of a tampered random seed should fail")
}

func TestSignAndVerifyRandomSeedOfMismatchedHeight(t *testing.T) {
	t.Skip("Remove the skip when block height is actually verified by VerifyRandomSeed()")
	keyPair := testKeys.EcdsaSecp256K1KeyPairForTests(0)
	mgr := NewKeyManager(log.DefaultTestingLogger(t), keyPair.PrivateKey())
	randomSeed := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	randomSeedSig := mgr.SignRandomSeed(1, randomSeed)
	senderSignature := lhprotocol.SenderSignatureBuilder{
		MemberId:  lhprimitives.MemberId(keyPair.NodeAddress()),
		Signature: lhprimitives.Signature(randomSeedSig),
	}
	err := mgr.VerifyRandomSeed(2, randomSeed, senderSignature.Build())
	require.Error(t, err, "Verification of random seed that was signed for another block height should fail")

}
