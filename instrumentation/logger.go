package instrumentation

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/scribe/log"
	"os"
)

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
