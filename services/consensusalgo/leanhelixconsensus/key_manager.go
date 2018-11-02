package leanhelixconsensus

import (
	"github.com/orbs-network/lean-helix-go"
	"github.com/orbs-network/lean-helix-go/primitives"
)

func (s *service) Sign(content []byte) []byte {
	panic("implement me")
}

func (s *service) Verify(content []byte, sender *leanhelix.SenderSignature) bool {
	panic("implement me")
}

func (s *service) MyPublicKey() primitives.Ed25519PublicKey {
	panic("implement me")
}
