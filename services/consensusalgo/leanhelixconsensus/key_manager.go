package leanhelixconsensus

import (
	"github.com/orbs-network/lean-helix-go"
	lhprimitives "github.com/orbs-network/lean-helix-go/primitives"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type keyManager struct {
	nodeAddress primitives.NodeAddress
	privateKey  primitives.EcdsaSecp256K1PrivateKey
	logger      log.BasicLogger
}

func NewKeyManager(logger log.BasicLogger, nodeAddress primitives.NodeAddress, privateKey primitives.EcdsaSecp256K1PrivateKey) *keyManager {
	return &keyManager{
		logger:      logger,
		nodeAddress: nodeAddress,
		privateKey:  privateKey,
	}
}

func (k *keyManager) Sign(content []byte) []byte {
	if len(content) == 0 {
		return []byte{} // TODO(v1): talkol added this because we must only sign 32 byte datas
	}
	sig, _ := signature.SignEcdsaSecp256K1(k.privateKey, content)
	return sig
}

func (k *keyManager) Verify(content []byte, sender *leanhelix.SenderSignature) bool {
	// TODO(v1): integrate this back inside
	// return digest.VerifyEcdsaSecp256K1WithNodeAddress(primitives.NodeAddress(sender.SenderPublicKey()), content, primitives.EcdsaSecp256K1Sig(sender.Signature()))
	return true
}

func (k *keyManager) MyPublicKey() lhprimitives.Ed25519PublicKey { // TODO(v1): change to node address, why does the library have knowledge of these Orbs-specific types??
	return lhprimitives.Ed25519PublicKey(k.nodeAddress)
}
