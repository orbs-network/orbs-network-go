// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package e2e

import (
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"net"
	"os"
)

type inProcessE2ENetwork struct {
	govnr.TreeSupervisor
	nodes          []*bootstrap.Node
	virtualChainId primitives.VirtualChainId
}

func NewLoggerRandomer() *loggerRandomer {
	console := log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter())
	logger := log.GetLogger().WithTags(
		log.String("_test", "e2e"),
		log.String("_branch", os.Getenv("GIT_BRANCH")),
		log.String("_commit", os.Getenv("GIT_COMMIT"))).
		WithOutput(console)
	tl := &loggerRandomer{logger: logger, console: console}
	rnd := rand.NewControlledRand(tl)
	tl.rnd = rnd
	// this is yuckie - it's a circular dependency, but it's ok since we're in a test situation and it's better than passing two arguments
	return tl
}

const firstEphemeralPort = 49152 // https://en.wikipedia.org/wiki/Ephemeral_port
const maxPort = 65535

type loggerRandomer struct {
	logger  log.Logger
	console log.Output
	rnd     *rand.ControlledRand
}

func (t *loggerRandomer) Log(args ...interface{}) {
	t.logger.Info(fmt.Sprintln(args...))
}

func (t *loggerRandomer) Name() string {
	return "e2e"
}

func (t *loggerRandomer) aRandomPort() int {
	for {
		port := firstEphemeralPort + t.rnd.Intn(maxPort-LOCAL_NETWORK_SIZE*2-firstEphemeralPort)
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			_ = l.Close()
			t.logger.Info("port is free, returning", log.Int("port", port))
			return port
		} else {
			t.logger.Info("port is already in use, retrying a different port", log.Int("port", port))
		}
	}
}
