package commands

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/ed25519"
	"os"
)

func HandleGenKeysCommand() {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	fmt.Println(hex.EncodeToString(publicKey))
	fmt.Println(hex.EncodeToString(privateKey))

	os.Exit(0)
}
