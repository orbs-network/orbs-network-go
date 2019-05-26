// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package signer

import (
	"github.com/orbs-network/orbs-network-go/crypto/signer"
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
	return signer.NewLocalSigner(s.config.NodePrivateKey()).Sign(payload)
}
