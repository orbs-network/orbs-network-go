package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/devtools/jsonapi"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"os"
)

// orbs-json-client [--send-transaction | --call-method]=<json> --public-key=<pubkey> --private-key=<privkey> --api-endpoint=<http://....>
func main() {
	sendTransactionPtr := flag.String("send-transaction", "", "<json>")
	callMethodPtr := flag.String("call-method", "", "<json>")
	verbosePtr := flag.Bool("v", false, "Show all related logs")

	//publicKeyPtr := flag.String("public-key")
	//privateKeyPtr := flag.String("private-key")

	apiEndpointPtr := flag.String("api-endpoint", "http://localhost:8080", "<http://..../api>")

	flag.Parse()

	logger := log.GetLogger(log.String("api-endpoint", *apiEndpointPtr)).WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

	if *sendTransactionPtr != "" {
		if *verbosePtr {
			logger.Info("sending transaction")
		}

		keyPair := keys.Ed25519KeyPairForTests(1)

		tx := &jsonapi.Transaction{}
		if err := json.Unmarshal([]byte(*sendTransactionPtr), tx); err != nil {
			logger.Error("could not unpack json", log.Error(err))
		}

		result, _ := jsonapi.SendTransaction(tx, keyPair, *apiEndpointPtr, *verbosePtr)

		jsonBytes, _ := json.Marshal(result.TransactionReceipt)
		fmt.Println(string(jsonBytes))
	} else if *callMethodPtr != "" {
		if *verbosePtr {
			logger.Info("calling method")
		}

		tx := &jsonapi.Transaction{}
		if err := json.Unmarshal([]byte(*callMethodPtr), tx); err != nil {
			logger.Error("could not unpack json", log.Error(err))
		}

		result, _ := jsonapi.CallMethod(tx, *apiEndpointPtr, *verbosePtr)

		jsonBytes, _ := json.Marshal(result)
		fmt.Println(string(jsonBytes))
	}
}
