package leanhelixconsensus

import (
	"github.com/orbs-network/lean-helix-go"
	lhprimitives "github.com/orbs-network/lean-helix-go/primitives"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

func (s *service) Sign(content []byte) []byte {
	sig, _ := signature.SignEd25519(s.config.NodePrivateKey(), content)
	return sig
}

func (s *service) Verify(content []byte, sender *leanhelix.SenderSignature) bool {

	return signature.VerifyEd25519(primitives.Ed25519PublicKey(sender.SenderPublicKey()), content, primitives.Ed25519Sig(sender.Signature()))
}

func (s *service) MyPublicKey() lhprimitives.Ed25519PublicKey {
	return lhprimitives.Ed25519PublicKey(s.config.NodePublicKey())
}
