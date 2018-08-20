package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/jsonapi"
	"os"
)

// orbs-json-client [--send-transaction | --call-method]=<json> --public-key=<pubkey> --private-key=<privkey> --api-endpoint=<http://....>
func main() {
	sendTransactionPtr := flag.String("send-transaction", "", "<json>")
	callMethodPtr := flag.String("call-method", "", "<json>")

	//publicKeyPtr := flag.String("public-key")
	//privateKeyPtr := flag.String("private-key")

	apiEndpointPtr := flag.String("api-endpoint", "http://localhost:8080/api", "<http://..../api>")

	flag.Parse()

	logger := log.GetLogger(log.String("api-endpoint", *apiEndpointPtr)).WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

	if *sendTransactionPtr != "" {
		logger.Info("sending transaction")

	} else if *callMethodPtr != "" {
		logger.Info("calling method")

		methodArgument := &jsonapi.Transaction{}
		if err := json.Unmarshal([]byte(*callMethodPtr), methodArgument); err != nil {
			logger.Error("could not unpack json", log.Error(err))
		}

		logger.Info("method argument", log.String("method-argument", fmt.Sprintf("%v", methodArgument)))

	}
}
