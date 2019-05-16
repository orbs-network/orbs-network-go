package main

import (
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/kms"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/scribe/log"
	"os"
)

func main() {
	httpAddress := flag.String("listen", ":7777", "ip address and port for http server")
	silentLog := flag.Bool("silent", false, "disable output to stdout")
	pathToLog := flag.String("log", "", "path/to/node.log")
	version := flag.Bool("version", false, "returns information about version")

	var configFiles config.ArrayFlags
	flag.Var(&configFiles, "config", "path/to/config.json")

	flag.Parse()

	if *version {
		fmt.Println(config.GetVersion())
		return
	}

	cfg, err := config.GetNodeConfigFromFiles(configFiles, *httpAddress)
	if err != nil {
		fmt.Printf("%s \n", err)
		os.Exit(1)
	}

	logger := instrumentation.GetLogger(*pathToLog, *silentLog, cfg).WithTags(log.Node(cfg.NodeAddress().String()))

	service := kms.NewService(cfg.HttpAddress(), cfg.NodePrivateKey(), logger)
	service.Start()
	service.WaitUntilShutdown()
}
