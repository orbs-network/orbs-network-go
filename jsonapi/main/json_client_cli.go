package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/jsonapi"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
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

		txBuilder := buildSignedTransaction(logger, []byte(*sendTransactionPtr))
		input := (&client.SendTransactionRequestBuilder{
			SignedTransaction: txBuilder,
		}).Build()

		callAPI(logger, *apiEndpointPtr, "send-transaction", input.Raw())
	} else if *callMethodPtr != "" {
		logger.Info("calling method")

		txBuilder := buildTransaction(logger, []byte(*callMethodPtr))
		input := (&client.CallMethodRequestBuilder{
			Transaction: txBuilder,
		}).Build()

		callAPI(logger, *apiEndpointPtr, "call-method", input.Raw())
	}
}

func buildSignedTransaction(logger log.BasicLogger, source []byte) *protocol.SignedTransactionBuilder {
	tx := &jsonapi.Transaction{}
	if err := json.Unmarshal(source, tx); err != nil {
		logger.Error("could not unpack json", log.Error(err))
	}

	logger.Info("method argument", log.String("method-argument", fmt.Sprintf("%v", tx)))
	keyPair := keys.Ed25519KeyPairForTests(1)

	signedTxBuilder, _ := jsonapi.ConvertAndSignTransaction(tx, keyPair)
	return signedTxBuilder
}

func buildTransaction(logger log.BasicLogger, source []byte) *protocol.TransactionBuilder {
	tx := &jsonapi.Transaction{}
	if err := json.Unmarshal(source, tx); err != nil {
		logger.Error("could not unpack json", log.Error(err))
	}

	logger.Info("method argument", log.String("method-argument", fmt.Sprintf("%v", tx)))
	return jsonapi.ConvertTransaction(tx)
}

func callAPI(logger log.BasicLogger, apiEndpoint string, apiMethod string, raw []byte) {
	bytes, err := httpPost(raw, apiEndpoint, apiMethod)
	output := jsonapi.ConvertCallMethodOutput(client.CallMethodResponseReader(bytes))

	if err != nil {
		logger.Error("api call error", log.Error(err))
	}

	logger.Info("received call method response", log.Stringable("result", output.CallResult), log.BlockHeight(output.BlockHeight), log.StringableSlice("output-args", output.OutputArguments))
}

func httpPost(raw []byte, apiEndpoint string, method string) ([]byte, error) {

	res, err := http.Post(apiEndpoint+method, "application/octet-stream", bytes.NewReader(raw))

	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	return bytes, nil
}
