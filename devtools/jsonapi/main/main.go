package main

import (
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/devtools/jsonapi/main/commands"
	"os"
)

// gamma-cli [--send-transaction | --call-method]=<json> --public-key=<pubkey> --private-key=<privkey> --api-endpoint=<http://....>
func main() {
	// Sub commands of the cli
	///runCommand := flag.NewFlagSet("run", flag.ExitOnError)
	//sendOperationPtr := runCommand.String("send-transaction", "", "<json>")
	//callMethodPtr := command.String("call-method", "", "<json>")
	//generateTestKeysPtr := command.Bool("generate-test-keys", false, "generates a pair of TEST keys, public and private; NEVER use them in production")
	//verbosePtr := command.Bool("v", false, "Show all related logs")
	//
	//publicKeyPtr := command.String("public-key", "", "public key in hex form")
	//privateKeyPtr := command.String("private-key", "", "public key in hex form")
	//
	//apiEndpointPtr := command.String("api-endpoint", "http://localhost:8080", "<http://..../api>")

	// Verify that a subcommand has been provided
	// os.Arg[0] is the main command
	// os.Arg[1] will be the subcommand itself
	if len(os.Args) < 2 {
		// TODO implement a welcome message here
		fmt.Println("must specify which command to run")
		os.Exit(1)
	}

	// Switch on the subcommand
	// Parse the flags for appropriate FlagSet
	// FlagSet.Parse() requires a set of arguments to parse as input
	// os.Args[2:] will be all arguments starting after the subcommand at os.Args[1]
	switch os.Args[1] {
	case "run":
		commands.HandleRunCommand(os.Args[2:])
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

	//logger := log.GetLogger(log.String("api-endpoint", *apiEndpointPtr)).WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	//
	//if *sendTransactionPtr != "" {
	//	if *verbosePtr {
	//		logger.Info("sending transaction")
	//	}
	//
	//	if *publicKeyPtr == "" || *privateKeyPtr == "" {
	//		logger.Error("Public key or private key is incorrect")
	//		return
	//	}
	//
	//	decodedPublicKey, publicKeyDecodeError := hex.DecodeString(*publicKeyPtr)
	//	decodedPrivateKey, privateKeyDecodeError := hex.DecodeString(*privateKeyPtr)
	//
	//	if publicKeyDecodeError != nil {
	//		logger.Error("Could not decode public key from HEX", log.Error(publicKeyDecodeError))
	//	}
	//
	//	if privateKeyDecodeError != nil {
	//		logger.Error("Could not decode private key from HEX", log.Error(privateKeyDecodeError))
	//	}
	//
	//	keyPair := keys.NewEd25519KeyPair(primitives.Ed25519PublicKey(decodedPublicKey), primitives.Ed25519PrivateKey(decodedPrivateKey))
	//
	//	tx := &jsonapi.Transaction{}
	//	if err := json.Unmarshal([]byte(*sendTransactionPtr), tx); err != nil {
	//		logger.Error("could not unpack json", log.Error(err))
	//	}
	//
	//	result, _ := jsonapi.SendTransaction(tx, keyPair, *apiEndpointPtr, *verbosePtr)
	//
	//	jsonBytes, _ := json.Marshal(result)
	//	fmt.Println(string(jsonBytes))
	//} else if *callMethodPtr != "" {
	//	if *verbosePtr {
	//		logger.Info("calling method")
	//	}
	//
	//	tx := &jsonapi.Transaction{}
	//	if err := json.Unmarshal([]byte(*callMethodPtr), tx); err != nil {
	//		logger.Error("could not unpack json", log.Error(err))
	//	}
	//
	//	result, _ := jsonapi.CallMethod(tx, *apiEndpointPtr, *verbosePtr)
	//
	//	jsonBytes, _ := json.Marshal(result)
	//	fmt.Println(string(jsonBytes))
	//} else if *generateTestKeysPtr {
	//	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	//
	//	fmt.Println(hex.EncodeToString(publicKey))
	//	fmt.Println(hex.EncodeToString(privateKey))
	//} else {
	//	flag.PrintDefaults()
	//}
}
