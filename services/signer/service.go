package signer

import (
	"github.com/orbs-network/orbs-network-go/crypto/kms"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
)

type Service interface {
	Sign([]byte) (primitives.EcdsaSecp256K1Sig, error)
}

type service struct {
	config ServiceConfig
	logger log.Logger
}

type ServiceConfig interface {
	NodePrivateKey() primitives.EcdsaSecp256K1PrivateKey
}

func NewService(config ServiceConfig, logger log.Logger) Service {
	return &service{
		config: config,
		logger: logger.WithTags(log.Service("signer")),
	}
}

func (s *service) Sign(payload []byte) (primitives.EcdsaSecp256K1Sig, error) {
	return kms.NewLocalSigner(s.config.NodePrivateKey()).Sign(payload)
}
