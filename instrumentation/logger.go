// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package instrumentation

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/scribe/log"
	"os"
)

func GetBootstrapCrashLogger() log.Logger {
	path := "./orbs-network-bootstrap.log"

	logFile, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	fileWriter := log.NewTruncatingFileWriter(logFile)
	outputs := []log.Output{
		log.NewFormattingOutput(fileWriter, log.NewHumanReadableFormatter()),
		log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()),
		log.NewFormattingOutput(os.Stderr, log.NewHumanReadableFormatter()),
	}

	return log.GetLogger().WithOutput(outputs...)
}

func GetLogger(path string, silent bool, cfg config.NodeConfig) log.Logger {
	if path == "" {
		path = "./orbs-network.log"
	}

	logFile, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	fileWriter := log.NewTruncatingFileWriter(logFile, cfg.LoggerFileTruncationInterval())
	outputs := []log.Output{
		log.NewFormattingOutput(fileWriter, log.NewJsonFormatter()),
	}

	if !silent {
		outputs = append(outputs, log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
	}

	if cfg.LoggerHttpEndpoint() != "" {
		customJSONFormatter := log.NewJsonFormatter().WithTimestampColumn("@timestamp")
		bulkSize := int(cfg.LoggerBulkSize())
		if bulkSize == 0 {
			bulkSize = 100
		}

		outputs = append(outputs, log.NewBulkOutput(log.NewHttpWriter(cfg.LoggerHttpEndpoint()), customJSONFormatter, bulkSize))
	}

	logger := log.GetLogger().WithTags(
		logfields.VirtualChainId(cfg.VirtualChainId()),
	).WithOutput(outputs...)

	conditionalFilter := log.NewConditionalFilter(false, nil)

	if !cfg.LoggerFullLog() {
		conditionalFilter = log.NewConditionalFilter(true, log.OnlyErrors())
	}

	return logger.WithFilters(conditionalFilter)
}
