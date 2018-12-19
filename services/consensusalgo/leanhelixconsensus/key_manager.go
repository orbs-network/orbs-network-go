package leanhelixconsensus

import (
	"encoding/binary"
	"github.com/orbs-network/lean-helix-go"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type keyManager struct {
	privateKey primitives.EcdsaSecp256K1PrivateKey
}

func (k *keyManager) SignConsensusMessage(blockHeight lhprimitives.BlockHeight, content []byte) []byte {
	sig, _ := digest.SignAsNode(k.privateKey, content) // TODO(v1): handle error (log) https://tree.taiga.io/project/orbs-network/us/603
	return sig
}

func (k *keyManager) VerifyConsensusMessage(blockHeight lhprimitives.BlockHeight, content []byte, signature lhprimitives.Signature, memberId lhprimitives.MemberId) bool {
	return digest.VerifyNodeSignature(primitives.NodeAddress(memberId), content, primitives.EcdsaSecp256K1Sig(signature))
}

func (k *keyManager) SignRandomSeed(blockHeight lhprimitives.BlockHeight, randomSeed []byte) []byte {
	sig, _ := digest.SignAsNode(k.privateKey, randomSeed) // TODO(v1): handle error (log) https://tree.taiga.io/project/orbs-network/us/603
	return sig
}

func (k *keyManager) VerifyRandomSeed(blockHeight lhprimitives.BlockHeight, randomSeed []byte, signature lhprimitives.Signature, memberId lhprimitives.MemberId) bool {
	return digest.VerifyNodeSignature(primitives.NodeAddress(memberId), randomSeed, primitives.EcdsaSecp256K1Sig(signature))
}

func (k *keyManager) AggregateRandomSeed(blockHeight lhprimitives.BlockHeight, randomSeedShares []*leanhelix.RandomSeedShare) lhprimitives.Signature {
	heightAsByteArray := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightAsByteArray, uint64(blockHeight))
	return lhprimitives.Signature(hash.CalcSha256(heightAsByteArray))
}

func NewKeyManager(privateKey primitives.EcdsaSecp256K1PrivateKey) *keyManager {
	return &keyManager{
		privateKey: privateKey,
	}
}
