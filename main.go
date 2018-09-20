package main

import (
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"os"
)

func getLogger(path string, silent bool) log.BasicLogger {
	if path == "" {
		path = "./orbs-network.log"
	}

	logFile, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	var stdout io.Writer
	stdout = os.Stdout

	if silent {
		stdout = ioutil.Discard
	}

	stdoutOutput := log.NewOutput(stdout).WithFormatter(log.NewHumanReadableFormatter())
	fileOutput := log.NewOutput(logFile).WithFormatter(log.NewHumanReadableFormatter()) // until system stabilizes we temporarily log files with human readable to assist debugging

	return log.GetLogger().WithOutput(stdoutOutput, fileOutput)
}

func getConfig(pathToConfig string) (config.NodeConfig, error) {
	cfg := config.ForProduction2()

	if pathToConfig != "" {
		if _, err := os.Stat(pathToConfig); os.IsNotExist(err) {
			return nil, errors.Errorf("could not open config file: %v", err)
		}

		contents, err := ioutil.ReadFile(pathToConfig)
		if err != nil {
			return nil, err
		}

		if mergedConfig, err := cfg.MergeWithFileConfig(string(contents)); err == nil {
			return mergedConfig, nil
		} else {
			return nil, err
		}
	}

	return cfg, nil
}

func main() {
	httpAddress := ":8080"

	silentLog := flag.Bool("silent", false, "disable output to stdout")
	pathToLog := flag.String("log", "", "path/to/node.log")
	pathToConfig := flag.String("config", "", "path/to/config.json")

	flag.Parse()

	cfg, err := getConfig(*pathToConfig)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	logger := getLogger(*pathToLog, *silentLog)

	bootstrap.NewNodeFromConfig(
		cfg,
		logger,
		httpAddress,
	).WaitUntilShutdown()
}
