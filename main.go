package main

import (
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
)

func getLogger(path string, silent bool, httpLogEndpoint string) log.BasicLogger {
	if path == "" {
		path = "./orbs-network.log"
	}

	logFile, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	outputs := []log.Output{
		log.NewFormattingOutput(logFile, log.NewJsonFormatter()),
	}

	if !silent {
		outputs = append(outputs, log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
	}

	if httpLogEndpoint != "" {
		customJSONFormatter := log.NewJsonFormatter().WithTimestampColumn("@timestamp")
		outputs = append(outputs, log.NewBulkOutput(log.NewHttpWriter(httpLogEndpoint), customJSONFormatter, 100))
	}

	return log.GetLogger().WithTags(
		log.String("_branch", os.Getenv("GIT_BRANCH")),
		log.String("_commit", os.Getenv("GIT_COMMIT")),
		log.String("_test", os.Getenv("TEST_NAME")),
	).WithOutput(outputs...)
}

func getConfig(configFiles config.ArrayFlags) (config.NodeConfig, error) {
	cfg := config.ForProduction("")

	if len(configFiles) != 0 {
		for _, configFile := range configFiles {
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				return nil, errors.Errorf("could not open config file: %v", err)
			}

			contents, err := ioutil.ReadFile(configFile)
			if err != nil {
				return nil, err
			}

			cfg, err = cfg.MergeWithFileConfig(string(contents))

			if err != nil {
				return nil, err
			}
		}
	}

	return cfg, nil
}

func main() {
	httpAddress := flag.String("listen", ":8080", "ip address and port for http server")
	silentLog := flag.Bool("silent", false, "disable output to stdout")
	httpLogEndpoint := flag.String("http-log-endpoint", "", "report logs to http/https endpoint (i.e. logz.io)")
	pathToLog := flag.String("log", "", "path/to/node.log")

	var configFiles config.ArrayFlags
	flag.Var(&configFiles, "config", "path/to/config.json")

	flag.Parse()

	cfg, err := getConfig(configFiles)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	logger := getLogger(*pathToLog, *silentLog, *httpLogEndpoint)

	bootstrap.NewNode(
		cfg,
		logger,
		*httpAddress,
	).WaitUntilShutdown()
}
