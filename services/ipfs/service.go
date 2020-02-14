// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package ipfs

import (
	"context"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/scribe/log"
)

var LogTag = log.Service("virtual-machine")

type service struct {
	logger log.Logger
	config config.NodeConfig // FIXME scale down
}

type IPFSReadInput struct {
}

type IPFSReadOutput struct {
}

type IPFSService interface {
	govnr.ShutdownWaiter
	Read(ctx context.Context, input *IPFSReadInput) (*IPFSReadOutput, error)
}

func NewIPFS(
	config config.NodeConfig,
	logger log.Logger,
) IPFSService {
	s := &service{
		logger: logger.WithTags(LogTag),
		config: config,
	}

	return s
}

func (s *service) Read(ctx context.Context, input *IPFSReadInput) (*IPFSReadOutput, error) {
	panic("implement me")
}

func (s *service) WaitUntilShutdown(shutdownContext context.Context) {
	s.logger.Info("shutting down")
}
