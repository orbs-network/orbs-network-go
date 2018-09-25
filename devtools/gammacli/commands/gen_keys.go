package commands

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"golang.org/x/crypto/ed25519"
)

func HandleGenKeysCommand() int {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Println("Could not generate a public/private key pair!", err)
		return 1
	}
	address := hash.CalcRipmd160Sha256(publicKey)

	fmt.Println(hex.EncodeToString(publicKey))
	fmt.Println(hex.EncodeToString(privateKey))
	fmt.Println(address)

	return 0
}
