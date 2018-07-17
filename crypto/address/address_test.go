package address_test

import (
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/address"
	"strconv"
	"testing"
)

type invalidTestPair struct {
	invalidAddress string
	testReason     string
}

var invalidAddressTests = []invalidTestPair{
	{"", "Empty address"},
	{"M00EXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4N3Qqa", "Too short"},
	{"M00EXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4N3Qq", "Too short"},
	{"M00EXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4N3Qqa1a", "Too long"},
	{"M00EXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4N3Qqa1aa", "Too long"},
	{"M00EXMPnnaWFqOyVxWdhYCgGzpnaL4qBy4N3Qqa1", "Invalid base58"},
	{"M00EXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4I3Qqa1", "Invalid base58"},
	{"M00EXMPnnaWFq0yVxWdhYCgGzpnaL4qBy4N3Qqa1", "Invalid base58"},
	{"M00EXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4N3Qqal", "Invalid base58"},
	{"M00EXMPnna+FqRyVxWdhYCgGzpnaL4qBy4N3Qqa1", "Invalid base58"},
	{"M00EXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4N3/qa1", "Invalid base58"},
	{"M00xKxXz7LPuyXmhxpoaNkr96jKrT99FsJ3AXQr", "Invalid vchain id"},
	{"M00H8exm1WU6CTGcpFiupsL7g1zN9dYoxMZ8ZrF", "Invalid vchain id"},
	{"300EXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4NZTSK1", "Invalid network id"},
	{"M05EXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4TZpGvu", "Invalid version"},
	{"M0FEXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4RjxghL", "Invalid version"},
	{"M00LUPVrDh4SDHggRBJHpT8hiBb6FEf2rMnxhySx", "Invalid checksum"},
	{"M00LUPVrDh4SDHggRBJHpT8hiBb6FEf2rMjMfiiL", "Invalid checksum"},
}

var validAddressTests = []string{
	"M00EXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4N3Qqa1",
	"T00EXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4TM9btp",
	"M00LUPVrDh4SDHggRBJHpT8hiBb6FEf2rMqZ9vza",
	"T00LUPVrDh4SDHggRBJHpT8hiBb6FEf2rMkGvQPR",
}

const (
	publicKey1 = "8d41d055d00459be37f749da2caf87bd4ced6fafa335b1f2142e0f44501b2c65"
	publicKey2 = "7a463487bb0eb584dabccd52398506b4a2dd432503cc6b7b582f87832ad104e6"
)

func testAE(actual, expected string) string {
	return fmt.Sprintf("a: %s, e: %s", actual, expected)
}

func TestAddressInitializationWithPublicKeyOnTestNet(t *testing.T) {
	pk2bytes, err := hex.DecodeString(publicKey2)
	if err != nil {
		t.Errorf("something went wrong with pk->bytes %s", err)
	}
	pktestNet, err := address.NewFromPK(pk2bytes, "9012ca", address.TEST_NETWORK_ID)
	if err != nil {
		t.Error(err)
	}
	if pktestNet.NetworkId() != address.TEST_NETWORK_ID {
		t.Errorf("address from pk on testnet, network id incorrect (%s)", testAE(pktestNet.NetworkId(), address.TEST_NETWORK_ID))
	}
	if pktestNet.VirtualChainId() != "9012ca" {
		t.Errorf("address from pk on testnet, vchain id incorrect (%s)", testAE(pktestNet.VirtualChainId(), "9012ca"))
	}
	if pktestNet.Version() != 0 {
		t.Errorf("address from pk on testnet, version is incorrect (%s)", testAE(string(pktestNet.Version()), "0"))
	}
	if accountId, err := pktestNet.AccountId(); err != nil {
		t.Error(err)
	} else if hex.EncodeToString(accountId) != "44068acc1b9ffc072694b684fc11ff229aff0b28" {
		t.Errorf("address from pk on testnet, account id is incorrect (%s)", testAE(hex.EncodeToString(accountId), "44068acc1b9ffc072694b684fc11ff229aff0b28"))
	}
	if raw, err := pktestNet.Raw(); err != nil {
		t.Error(err)
	} else if address.ToBase58(raw) != "T00LUPVrDh4SDHggRBJHpT8hiBb6FEf2rMkGvQPR" {
		t.Errorf("address from pk on testnet, base58 is incorrect (%s)", testAE(string(address.ToBase58(raw)), "T00LUPVrDh4SDHggRBJHpT8hiBb6FEf2rMkGvQPR"))
	}
	// if the above is okay, then the checksum must be okay..
	if checksum, err := pktestNet.Checksum(); err != nil {
		t.Error(err)
	} else if checksum != 0x258c93e8 {
		t.Errorf("address from pk on testnet, checksum is incorrect (%s)", testAE(strconv.FormatUint(uint64(checksum), 16), strconv.FormatUint(0x258c93e8, 16)))
	}
}

func TestAddressInitializationWithKeyOnMainNet(t *testing.T) {
	pk1bytes, err := hex.DecodeString(publicKey1)
	if err != nil {
		t.Errorf("something went wrong with pk->bytes %s", err)
	}
	pktestNet, err := address.NewFromPK(pk1bytes, "640ed3", address.MAIN_NETWORK_ID)
	if err != nil {
		t.Error(err)
	}
	if pktestNet.NetworkId() != address.MAIN_NETWORK_ID {
		t.Errorf("address from pk on testnet, network id incorrect (%s)", testAE(pktestNet.NetworkId(), address.MAIN_NETWORK_ID))
	}
	if pktestNet.VirtualChainId() != "640ed3" {
		t.Errorf("address from pk on testnet, vchain id incorrect (%s)", testAE(pktestNet.VirtualChainId(), "640ed3"))
	}
	if pktestNet.Version() != 0 {
		t.Errorf("address from pk on testnet, version is incorrect (%s)", testAE(string(pktestNet.Version()), "0"))
	}
	if accountId, err := pktestNet.AccountId(); err != nil {
		t.Error(err)
	} else if hex.EncodeToString(accountId) != "c13052d8208230a58ab363708c08e78f1125f488" {
		t.Errorf("address from pk on testnet, account id is incorrect (%s)", testAE(hex.EncodeToString(accountId), "c13052d8208230a58ab363708c08e78f1125f488"))
	}
	if raw, err := pktestNet.Raw(); err != nil {
		t.Error(err)
	} else if address.ToBase58(raw) != "M00EXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4N3Qqa1" {
		t.Errorf("address from pk on testnet, base58 is incorrect (%s)", testAE(string(address.ToBase58(raw)), "M00EXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4N3Qqa1"))
	}
	// if the above is okay, then the checksum must be okay..
	if checksum, err := pktestNet.Checksum(); err != nil {
		t.Error(err)
	} else if checksum != 0xb4af4d2 {
		t.Errorf("address from pk on testnet, checksum is incorrect (%s)", testAE(strconv.FormatUint(uint64(checksum), 16), strconv.FormatUint(0xb4af4d2, 16)))
	}
}

func TestAddressInitializationFailsOnInvalidPK(t *testing.T) {
	t.Error("need to implement")
}

func TestAddressInitializationFailsOnInvalidVChainId(t *testing.T) {
	t.Error("need to implement")
}

func TestAddressInitializationFailsOnInvalidNetworkId(t *testing.T) {
	t.Error("need to implement")
}

func TestAddressSerialization(t *testing.T) {
	t.Error("need to implement")
}

func TestAddressSerializationFailsOnIncorrectChecksum(t *testing.T) {
	t.Error("need to implement")
}

func TestAddressSerializationFailsOnPKMismatch(t *testing.T) {
	t.Error("need to implement")
}

func TestAddressIsValid(t *testing.T) {
	for _, test := range validAddressTests {
		if r, err := address.IsValid(test); err != nil || !r {
			t.Errorf("address %s should be valid but is not: %s", test, err)
		}
	}
}

func TestAddressIsValidFails(t *testing.T) {
	for _, pair := range invalidAddressTests {
		if _, err := address.IsValid(pair.invalidAddress); err == nil {
			t.Errorf("invalid address passed validation: %s", pair.invalidAddress)
		}
	}
}
