package commands

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/devtools/gammacli"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"io/ioutil"
	"os"
	"strings"
)

func ShowUsage() {
	fmt.Println("Usage:  $ gamma-cli run send path/to/send.json")
	fmt.Println("Usage:  $ gamma-cli run call path/to/get.json")
}

func getKeypairFromOrbsKeyFile() (*keys.Ed25519KeyPair, error) {
	keyFile := "./.orbsKeys"
	if _, err := os.Stat(keyFile); err != nil {
		fmt.Println("Could not find a .orbsKeys file in the current directory")
		return nil, err
	}

	keyFileBytes, err := ioutil.ReadFile(keyFile)
	if err != nil {
		fmt.Println("Could not open key file", err)
		return nil, err
	}

	keysExploded := strings.Split(string(keyFileBytes), "\n")

	publicKey, err := hex.DecodeString(keysExploded[0])
	if err != nil {
		fmt.Println("Could not decode public key from hex", err)
		return nil, err
	}
	privateKey, err := hex.DecodeString(keysExploded[1])
	if err != nil {
		fmt.Println("Could not decode private key from hex", err)
		return nil, err
	}

	keyPair := keys.NewEd25519KeyPair(primitives.Ed25519PublicKey(publicKey), primitives.Ed25519PrivateKey(privateKey))

	return keyPair, nil
}

func getKeypairFromFlags(publicKey string, privateKey string) (*keys.Ed25519KeyPair, error) {
	decodedPublicKey, publicKeyDecodeError := hex.DecodeString(publicKey)
	decodedPrivateKey, privateKeyDecodeError := hex.DecodeString(privateKey)

	if publicKeyDecodeError != nil {
		fmt.Println("Could not decode public key from HEX", publicKeyDecodeError)
		return nil, publicKeyDecodeError
	}

	if privateKeyDecodeError != nil {
		fmt.Println("Could not decode private key from HEX", privateKeyDecodeError)
		return nil, privateKeyDecodeError
	}

	keyPair := keys.NewEd25519KeyPair(primitives.Ed25519PublicKey(decodedPublicKey), primitives.Ed25519PrivateKey(decodedPrivateKey))

	return keyPair, nil
}

func HandleRunCommand(args []string) int {
	if len(args) < 2 {
		ShowUsage()
		return 1
	}

	flagSet := flag.NewFlagSet("run", flag.ExitOnError)

	publicKeyPtr := flagSet.String("public-key", "", "public key in hex form")
	privateKeyPtr := flagSet.String("private-key", "", "public key in hex form")
	hostPtr := flagSet.String("host", "http://localhost:8080", "<http://..../api>")

	flagSet.Parse(args[2:])

	runType := args[0]
	pathToJson := args[1]

	tx := &gammacli.JSONTransaction{}
	var jsonBytes []byte
	_, err := os.Stat(pathToJson)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	if err == nil {
		jsonBytes, err = ioutil.ReadFile(pathToJson)

		if err != nil {
			fmt.Println("Could not open JSON file", err)
			return 1
		}

		if err := json.Unmarshal(jsonBytes, tx); err != nil {
			fmt.Println("could not parse JSON", err)
			return 1
		}
	}

	switch runType {
	case "send":
		var keyPair *keys.Ed25519KeyPair

		if *publicKeyPtr != "" && *privateKeyPtr != "" {
			keyPair, err = getKeypairFromFlags(*publicKeyPtr, *privateKeyPtr)
		} else {
			keyPair, err = getKeypairFromOrbsKeyFile()
		}

		if err != nil {
			return 1
		}

		result, err := gammacli.SendTransaction(tx, keyPair, *hostPtr, false)
		if err != nil {
			fmt.Println("Error sending your transaction", err)
			return 1
		}

		jsonBytes, _ := json.Marshal(result)
		fmt.Println(string(jsonBytes))
		return 0
	case "call":
		result, _ := gammacli.CallMethod(tx, *hostPtr, false)

		jsonBytes, _ := json.Marshal(result)
		fmt.Println(string(jsonBytes))
		return 0
	default:
		ShowUsage()
	}
	return 1
}
