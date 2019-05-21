package main

import (
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap/signer"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/scribe/log"
	"os"
)

func getLogger() log.Logger {
	return log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
}

func main() {
	httpAddress := flag.String("listen", ":7777", "ip address and port for http server")
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

	signer.StartSignerServer(cfg, getLogger()).WaitUntilShutdown()
}
