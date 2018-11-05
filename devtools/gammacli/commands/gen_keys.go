package commands

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"golang.org/x/crypto/ed25519"
)

func (r *CommandRunner) HandleGenKeysCommand() (string, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "Could not generate a public/private key pair!", err
	}
	address := hash.CalcRipmd160Sha256(publicKey)

	returnString := string(hex.EncodeToString(publicKey)) + "\n" +
		string(hex.EncodeToString(privateKey)) + "\n" +
		string(address)

	return returnString, nil
}
