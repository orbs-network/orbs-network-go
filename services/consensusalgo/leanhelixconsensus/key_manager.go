package leanhelixconsensus

import (
	"encoding/binary"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	lhprotocol "github.com/orbs-network/lean-helix-go/spec/types/go/protocol"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type keyManager struct {
	privateKey primitives.EcdsaSecp256K1PrivateKey
}

func (k *keyManager) SignRandomSeed(blockHeight lhprimitives.BlockHeight, content []byte) lhprimitives.RandomSeedSignature {
	sig, _ := digest.SignAsNode(k.privateKey, content) // TODO(v1): handle error (log) https://tree.taiga.io/project/orbs-network/us/603
	return lhprimitives.RandomSeedSignature(sig)
}

func (k *keyManager) SignConsensusMessage(blockHeight lhprimitives.BlockHeight, content []byte) lhprimitives.Signature {
	sig, _ := digest.SignAsNode(k.privateKey, content) // TODO(v1): handle error (log) https://tree.taiga.io/project/orbs-network/us/603
	return lhprimitives.Signature(sig)
}

func (k *keyManager) VerifyConsensusMessage(blockHeight lhprimitives.BlockHeight, content []byte, sender *lhprotocol.SenderSignature) bool {
	return digest.VerifyNodeSignature(primitives.NodeAddress(sender.MemberId()), content, primitives.EcdsaSecp256K1Sig(sender.Signature()))
}

func (k *keyManager) VerifyRandomSeed(blockHeight lhprimitives.BlockHeight, content []byte, sender *lhprotocol.SenderSignature) bool {
	return digest.VerifyNodeSignature(primitives.NodeAddress(sender.MemberId()), content, primitives.EcdsaSecp256K1Sig(sender.Signature()))
}

func (k *keyManager) AggregateRandomSeed(blockHeight lhprimitives.BlockHeight, randomSeedShares []*lhprotocol.SenderSignature) lhprimitives.RandomSeedSignature {
	heightAsByteArray := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightAsByteArray, uint64(blockHeight))
	return lhprimitives.RandomSeedSignature(hash.CalcSha256(heightAsByteArray))
}

func NewKeyManager(privateKey primitives.EcdsaSecp256K1PrivateKey) *keyManager {
	return &keyManager{
		privateKey: privateKey,
	}
}
