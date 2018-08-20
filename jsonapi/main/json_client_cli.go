package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/jsonapi"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"io/ioutil"
	"net/http"
	"os"
)

// orbs-json-client [--send-transaction | --call-method]=<json> --public-key=<pubkey> --private-key=<privkey> --api-endpoint=<http://....>
func main() {
	sendTransactionPtr := flag.String("send-transaction", "", "<json>")
	callMethodPtr := flag.String("call-method", "", "<json>")

	//publicKeyPtr := flag.String("public-key")
	//privateKeyPtr := flag.String("private-key")

	apiEndpointPtr := flag.String("api-endpoint", "http://localhost:8080/api/", "<http://..../api>")

	flag.Parse()

	logger := log.GetLogger(log.String("api-endpoint", *apiEndpointPtr)).WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

	if *sendTransactionPtr != "" {
		logger.Info("sending transaction")

	} else if *callMethodPtr != "" {
		logger.Info("calling method")

		tx := &jsonapi.Transaction{}
		if err := json.Unmarshal([]byte(*callMethodPtr), tx); err != nil {
			logger.Error("could not unpack json", log.Error(err))
		}

		logger.Info("method argument", log.String("method-argument", fmt.Sprintf("%v", tx)))
		keyPair := keys.Ed25519KeyPairForTests(1)

		signedTxBuilder, _ := jsonapi.ConvertAndSignTransaction(tx, keyPair)

		response, err := httpPost(signedTxBuilder.Build().Raw(), *apiEndpointPtr, "call-method")

		if err != nil {
			logger.Error("api call error", log.Error(err))
		}

		logger.Info("received call method response", log.Stringable("response", response))
	}
}

func httpPost(raw []byte, apiEndpoint string, method string) (*services.CallMethodOutput, error) {
	res, err := http.Post(apiEndpoint+method, "application/octet-stream", bytes.NewReader(raw))

	if err == nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	return &services.CallMethodOutput{ClientResponse: client.CallMethodResponseReader(bytes)}, nil
}
