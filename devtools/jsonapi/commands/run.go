package commands

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/devtools/jsonapi"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"io/ioutil"
	"os"
	"strings"
)

func ShowUsage() {
	fmt.Println("Usage:  $ gamma-cli run send path/to/send.json")
	fmt.Println("Usage:  $ gamma-cli run call path/to/get.json")
	os.Exit(2)
}

func getKeypairFromOrbsKeyFile() *keys.Ed25519KeyPair {
	keyFile := "./.orbsKeys"
	if _, err := os.Stat(keyFile); err != nil {
		fmt.Println("Could not find a .orbsKeys file in the current directory")
		os.Exit(2)
	}

	keyFileBytes, err := ioutil.ReadFile(keyFile)
	if err != nil {
		fmt.Println("Could not open key file", err)
		os.Exit(1)
	}

	keysExploded := strings.Split(string(keyFileBytes), "\n")

	publicKey, err := hex.DecodeString(keysExploded[0])
	if err != nil {
		fmt.Println("Could not decode public key from hex", err)
		os.Exit(1)
	}
	privateKey, err := hex.DecodeString(keysExploded[1])
	if err != nil {
		fmt.Println("Could not decode private key from hex", err)
		os.Exit(1)
	}

	keyPair := keys.NewEd25519KeyPair(primitives.Ed25519PublicKey(publicKey), primitives.Ed25519PrivateKey(privateKey))

	return keyPair
}

func getKeypairFromFlags(publicKey string, privateKey string) *keys.Ed25519KeyPair {
	decodedPublicKey, publicKeyDecodeError := hex.DecodeString(publicKey)
	decodedPrivateKey, privateKeyDecodeError := hex.DecodeString(privateKey)

	if publicKeyDecodeError != nil {
		fmt.Println("Could not decode public key from HEX", publicKeyDecodeError)
		os.Exit(1)
	}

	if privateKeyDecodeError != nil {
		fmt.Println("Could not decode private key from HEX", privateKeyDecodeError)
		os.Exit(1)
	}

	keyPair := keys.NewEd25519KeyPair(primitives.Ed25519PublicKey(decodedPublicKey), primitives.Ed25519PrivateKey(decodedPrivateKey))

	return keyPair
}

func fixInputNumbers(tx *jsonapi.Transaction) {
	for _, arg := range tx.Arguments {
		if arg.Type == "uint64" {
			arg.Value = uint64(arg.Value.(float64))
		}

		if arg.Type == "uint32" {
			arg.Value = uint32(arg.Value.(float64))
		}
	}
}

func HandleRunCommand(args []string) {
	if len(args) < 2 {
		ShowUsage()
	}

	flagSet := flag.NewFlagSet("run", flag.ExitOnError)

	publicKeyPtr := flagSet.String("public-key", "", "public key in hex form")
	privateKeyPtr := flagSet.String("private-key", "", "public key in hex form")
	hostPtr := flagSet.String("host", "http://localhost:8080", "<http://..../api>")

	flagSet.Parse(args[2:])

	runType := args[0]
	pathToJson := args[1]

	tx := &jsonapi.Transaction{}
	var jsonBytes []byte
	_, err := os.Stat(pathToJson)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err == nil {
		jsonBytes, err = ioutil.ReadFile(pathToJson)

		if err != nil {
			fmt.Println("Could not open JSON file", err)
			os.Exit(1)
		}

		if err := json.Unmarshal(jsonBytes, tx); err != nil {
			fmt.Println("could not parse JSON", err)
		}
	}

	switch runType {
	case "send":
		var keyPair *keys.Ed25519KeyPair

		if *publicKeyPtr != "" && *privateKeyPtr != "" {
			keyPair = getKeypairFromFlags(*publicKeyPtr, *privateKeyPtr)
		} else {
			keyPair = getKeypairFromOrbsKeyFile()
		}

		result, err := jsonapi.SendTransaction(tx, keyPair, *hostPtr, false)
		if err != nil {
			fmt.Println("Error sending your transaction", err)
			os.Exit(1)
		}

		jsonBytes, _ := json.Marshal(result)
		fmt.Println(string(jsonBytes))
		os.Exit(0)
	case "call":
		result, _ := jsonapi.CallMethod(tx, *hostPtr, false)

		jsonBytes, _ := json.Marshal(result)
		fmt.Println(string(jsonBytes))
		os.Exit(0)
	default:
		ShowUsage()
	}
}
