package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/devtools/jsonapi"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"golang.org/x/crypto/ed25519"
	"os"
)

// orbs-json-client [--send-transaction | --call-method]=<json> --public-key=<pubkey> --private-key=<privkey> --api-endpoint=<http://....>
func main() {
	sendTransactionPtr := flag.String("send-transaction", "", "<json>")
	callMethodPtr := flag.String("call-method", "", "<json>")
	generateTestKeysPtr := flag.Bool("generate-test-keys", false, "generates a pair of TEST keys, public and private; NEVER use them in production")
	verbosePtr := flag.Bool("v", false, "Show all related logs")

	publicKeyPtr := flag.String("public-key", "", "public key in hex form")
	privateKeyPtr := flag.String("private-key", "", "public key in hex form")

	apiEndpointPtr := flag.String("api-endpoint", "http://localhost:8080", "<http://..../api>")

	flag.Parse()

	logger := log.GetLogger(log.String("api-endpoint", *apiEndpointPtr)).WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

	if *sendTransactionPtr != "" {
		if *verbosePtr {
			logger.Info("sending transaction")
		}

		if *publicKeyPtr == "" || *privateKeyPtr == "" {
			logger.Error("Public key or private key is incorrect")
			return
		}

		decodedPublicKey, publicKeyDecodeError := hex.DecodeString(*publicKeyPtr)
		decodedPrivateKey, privateKeyDecodeError := hex.DecodeString(*privateKeyPtr)

		if publicKeyDecodeError != nil {
			logger.Error("Could not decode public key from HEX", log.Error(publicKeyDecodeError))
		}

		if privateKeyDecodeError != nil {
			logger.Error("Could not decode private key from HEX", log.Error(privateKeyDecodeError))
		}

		keyPair := keys.NewEd25519KeyPair(primitives.Ed25519PublicKey(decodedPublicKey), primitives.Ed25519PrivateKey(decodedPrivateKey))

		tx := &jsonapi.Transaction{}
		if err := json.Unmarshal([]byte(*sendTransactionPtr), tx); err != nil {
			logger.Error("could not unpack json", log.Error(err))
		}

		result, _ := jsonapi.SendTransaction(tx, keyPair, *apiEndpointPtr, *verbosePtr)

		jsonBytes, _ := json.Marshal(result)
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
	} else if *generateTestKeysPtr {
		publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

		fmt.Println(hex.EncodeToString(publicKey))
		fmt.Println(hex.EncodeToString(privateKey))
	} else {
		flag.PrintDefaults()
	}
}
