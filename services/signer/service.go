// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package signer

import (
	"context"
	"github.com/orbs-network/crypto-lib-go/crypto/signer"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
)

type service struct {
	config ServiceConfig
	logger log.Logger
}

type ServiceConfig interface {
	NodePrivateKey() primitives.EcdsaSecp256K1PrivateKey
}

func NewService(config ServiceConfig, logger log.Logger) services.Vault {
	return &service{
		config: config,
		logger: logger.WithTags(log.Service("signer")),
	}
}

func (s *service) NodeSign(ctx context.Context, input *services.NodeSignInput) (*services.NodeSignOutput, error) {
	signature, err := signer.NewLocalSigner(s.config.NodePrivateKey()).Sign(ctx, input.Data())
	if err != nil {
		s.logger.Error("Node sign error", log.Error(err))
		return nil, err
	}

	return (&services.NodeSignOutputBuilder{
		Signature: signature,
	}).Build(), nil
}
