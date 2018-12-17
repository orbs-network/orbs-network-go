package leanhelixconsensus

import (
	lhprotocol "github.com/orbs-network/lean-helix-go/spec/types/go/protocol"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type keyManager struct {
	privateKey primitives.Ed25519PrivateKey
	logger     log.BasicLogger
}

func NewKeyManager(logger log.BasicLogger, privateKey primitives.Ed25519PrivateKey) *keyManager {
	return &keyManager{
		logger:     logger,
		privateKey: privateKey,
	}
}

func (k *keyManager) Sign(content []byte) []byte {
	sig, _ := signature.SignEd25519(k.privateKey, content)
	return sig
}

func (k *keyManager) Verify(content []byte, sender *lhprotocol.SenderSignature) bool {

	return signature.VerifyEd25519(primitives.Ed25519PublicKey(sender.MemberId()), content, primitives.Ed25519Sig(sender.Signature()))
}
