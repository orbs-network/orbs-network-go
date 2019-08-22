// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package signer

import (
	"context"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/services/signer"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
)

type Server struct {
	govnr.TreeSupervisor
	service    services.Vault
	cancelFunc context.CancelFunc
	httpServer *httpServer
}

type ServerConfig interface {
	NodePrivateKey() primitives.EcdsaSecp256K1PrivateKey
	HttpAddress() string
}

func StartSignerServer(cfg ServerConfig, logger log.Logger) *Server {
	_, cancel := context.WithCancel(context.Background())

	service := signer.NewService(cfg, logger)
	api := &api{
		service, logger,
	}

	httpServer, err := NewHttpServer(cfg.HttpAddress(), logger)
	// Must find a better way
	if err != nil {
		panic(err)
	}

	httpServer.Router().HandleFunc("/sign", api.SignHandler)

	s := &Server{
		service:    service,
		cancelFunc: cancel,
		httpServer: httpServer,
	}

	s.Supervise(httpServer)

	return s
}

func (s *Server) GracefulShutdown(shutdownContext context.Context) {
	supervised.ShutdownAllGracefully(shutdownContext, s.httpServer)
}
