package leanhelixconsensus

import (
	lhprotocol "github.com/orbs-network/lean-helix-go/spec/types/go/protocol"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type keyManager struct {
	privateKey primitives.EcdsaSecp256K1PrivateKey
	logger     log.BasicLogger
}

func NewKeyManager(logger log.BasicLogger, privateKey primitives.EcdsaSecp256K1PrivateKey) *keyManager {
	return &keyManager{
		logger:     logger,
		privateKey: privateKey,
	}
}

func (k *keyManager) Sign(content []byte) []byte {
	sig, _ := digest.SignAsNode(k.privateKey, content) // TODO(v1): handle error (log)
	return sig
}

func (k *keyManager) Verify(content []byte, sender *lhprotocol.SenderSignature) bool {
	return digest.VerifyNodeSignature(primitives.NodeAddress(sender.MemberId()), content, primitives.EcdsaSecp256K1Sig(sender.Signature()))
}
