// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package config

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

type ArrayFlags []string

func (i *ArrayFlags) String() string {
	return "my string representation"
}

func (i *ArrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type filePollingLoader struct {
	sync.Mutex
	files              []string
	handlers           []ChangeHandler
	contentsForPolling map[string]primitives.Sha256
}

func NewFilePollingLoader(configFiles ...string) *filePollingLoader {
	l := &filePollingLoader{files: configFiles}
	l.contentsForPolling = make(map[string]primitives.Sha256)
	return l
}

func (l *filePollingLoader) Load() (*MapBasedConfig, error) {
	cfg := ForProduction("")

	for _, configFile := range l.files {
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			return nil, errors.Wrapf(err, "could not open config file: %s", configFile)
		}

		contents, err := ioutil.ReadFile(configFile)
		if err != nil {
			return nil, errors.Wrapf(err, "could not read config file: %s", configFile)
		}

		l.storeContents(configFile, contents)

		cfg, err = cfg.MergeWithJSONConfig(string(contents))

		if err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func (l *filePollingLoader) OnConfigChanged(handler ChangeHandler) {
	l.Lock()
	defer l.Unlock()
	l.handlers = append(l.handlers, handler)
}

func (l *filePollingLoader) ListenForChanges(ctx context.Context, logger log.Logger, pollInterval time.Duration, onShutdown func()) {
	synchronization.NewPeriodicalTrigger(ctx, pollInterval, logger, func() {
		if err := l.pollForChangesAndMaybeNotify(); err != nil {
			logger.Error("failed polling for config changes", log.Error(err))
		}
	}, onShutdown)
}

func (l *filePollingLoader) pollForChangesAndMaybeNotify() error {
	for _, configFile := range l.files {
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			return errors.Wrapf(err, "could not open config file: %s", configFile)
		}

		newContents, err := ioutil.ReadFile(configFile)
		if err != nil {
			return errors.Wrapf(err, "could not read config file: %s", configFile)
		}

		if l.hasChanged(configFile, newContents) {
			if newCfg, err := l.Load(); err != nil {
				return err
			} else {
				l.notifyHandlers(newCfg)
			}
		}
	}

	return nil
}

func (l *filePollingLoader) notifyHandlers(newConfig *MapBasedConfig) {
	l.Lock()
	defer l.Unlock()
	for i := range l.handlers {
		handler := l.handlers[i]
		handler(newConfig)
	}
}

func (l *filePollingLoader) hasChanged(fileName string, newContents []byte) bool {
	l.Lock()
	defer l.Unlock()
	return !l.contentsForPolling[fileName].Equal(hash.CalcSha256(newContents))
}

func (l *filePollingLoader) storeContents(fileName string, newContent []byte) {
	l.Lock()
	defer l.Unlock()
	l.contentsForPolling[fileName] = hash.CalcSha256(newContent)
}
