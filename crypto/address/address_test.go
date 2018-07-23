package address_test

import (
	"bytes"
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

func pkStringToBytes(t *testing.T, pk string) []byte {
	pk1bytes, err := hex.DecodeString(pk)
	if err != nil {
		t.Errorf("something went wrong with pk->bytes %s", err)
	}
	return pk1bytes
}

func validateAddressFromPK(t *testing.T, a *address.Address, net, vchain, aid, rawbs58 string, version uint8, checksum uint32) {
	if a.NetworkId() != net {
		t.Errorf("address from pk, network id incorrect (%s)", testAE(a.NetworkId(), net))
	}
	if a.VirtualChainId() != vchain {
		t.Errorf("address from pk, vchain id incorrect (%s)", testAE(a.VirtualChainId(), vchain))
	}
	if a.Version() != version {
		t.Errorf("address from pk, version is incorrect (%s)", testAE(string(a.Version()), strconv.Itoa(int(version))))
	}
	if accountId, err := a.AccountId(); err != nil {
		t.Error(err)
	} else if hex.EncodeToString(accountId) != aid {
		t.Errorf("address from pk, account id is incorrect (%s)", testAE(hex.EncodeToString(accountId), aid))
	}
	if raw, err := a.Raw(); err != nil {
		t.Error(err)
	} else if address.Base58Encode(raw) != rawbs58 {
		t.Errorf("address from pk, base58 is incorrect (%s)", testAE(string(address.Base58Encode(raw)), rawbs58))
	}
	// if the above is okay, then the checksum must be okay..
	if cs, err := a.Checksum(); err != nil {
		t.Error(err)
	} else if cs != checksum {
		t.Errorf("address from pk, checksum is incorrect (%s)", testAE(strconv.FormatUint(uint64(cs), 16), strconv.FormatUint(uint64(checksum), 16)))
	}
}

func TestAddressInitializationWithPublicKeyOnTestNet(t *testing.T) {
	pk2bytes := pkStringToBytes(t, publicKey2)

	pktestNet, err := address.NewFromPK(pk2bytes, "9012ca", address.TEST_NETWORK_ID)
	if err != nil {
		t.Fatal(err)
	}
	validateAddressFromPK(
		t,
		pktestNet,
		address.TEST_NETWORK_ID,
		"9012ca",
		"44068acc1b9ffc072694b684fc11ff229aff0b28",
		"T00LUPVrDh4SDHggRBJHpT8hiBb6FEf2rMkGvQPR",
		0,
		0x258c93e8)
}

func TestAddressInitializationWithKeyOnMainNet(t *testing.T) {
	pk1bytes := pkStringToBytes(t, publicKey1)
	pkmainNet, err := address.NewFromPK(pk1bytes, "640ed3", address.MAIN_NETWORK_ID)
	if err != nil {
		t.Fatal(err)
	}
	validateAddressFromPK(
		t,
		pkmainNet,
		address.MAIN_NETWORK_ID,
		"640ed3",
		"c13052d8208230a58ab363708c08e78f1125f488",
		"M00EXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4N3Qqa1",
		0,
		0xb4af4d2)
}

func TestAddressInitializationFailsOnInvalidPK(t *testing.T) {
	_, err := address.NewFromPK([]byte{}, "010101", address.TEST_NETWORK_ID)
	if err == nil {
		t.Error("address initialized without pk")
	}
}

func TestAddressInitializationFailsOnInvalidVChainIdHex(t *testing.T) {
	pk1bytes := pkStringToBytes(t, publicKey1)
	if _, err := address.NewFromPK(pk1bytes, "1", address.TEST_NETWORK_ID); err == nil {
		t.Error("address initialized on invalid virtual chain id")
	}
}

func TestAddressInitializationFailsOnInvalidVChainId(t *testing.T) {
	pk1bytes := pkStringToBytes(t, publicKey1)
	if _, err := address.NewFromPK(pk1bytes, "010101", address.TEST_NETWORK_ID); err == nil {
		t.Error("address initialized on invalid virtual chain id")
	}
}

func TestAddressInitializationFailsOnInvalidNetworkId(t *testing.T) {
	pk1bytes := pkStringToBytes(t, publicKey1)
	if _, err := address.NewFromPK(pk1bytes, "101010", "Z"); err == nil {
		t.Error("address initialized on invalid network id")
	}
}

func TestAddressInitializationFailsOnInvalidVersion(t *testing.T) {
	pk1bytes := pkStringToBytes(t, publicKey1)
	if _, err := address.NewFromAddress("M0FEXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4N3Qqa1", pk1bytes); err == nil {
		t.Error("address initialized on invalid version")
	}
}

func TestAddressSerialization(t *testing.T) {
	pk1bytes := pkStringToBytes(t, publicKey1)
	pkmainNet, err := address.NewFromPK(pk1bytes, "640ed3", address.MAIN_NETWORK_ID)
	if err != nil {
		t.Fatalf("failed to generate new address (unstable): %s", err)
	}

	dsraw, err := address.Base58Decode("M00EXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4N3Qqa1")
	if err != nil {
		t.Fatalf("failed to decode: %s", err)
	}

	if raw, err := pkmainNet.Raw(); err != nil {
		t.Errorf("failed to generate raw from address (unstable)")
	} else {
		if !bytes.Equal(dsraw, raw) {
			t.Errorf("deserialization failed, raw deserialized does not match expcted address")
		}
	}
}

func TestAddressSerializationFailsOnIncorrectChecksum(t *testing.T) {
	pk2bytes := pkStringToBytes(t, publicKey2)

	if _, err := address.NewFromAddress("M00LUPVrDh4SDHggRBJHpT8hiBb6FEf2rMqZ9vya", pk2bytes); err == nil {
		t.Errorf("invalid checksum and deserialization worked: %s", err)
	}
}

func TestAddressSerializationFailsOnPKMismatch(t *testing.T) {
	pk2bytes := pkStringToBytes(t, publicKey1)

	if a, err := address.NewFromAddress("M00LUPVrDh4SDHggRBJHpT8hiBb6FEf2rMqZ9vza", pk2bytes); err == nil {
		t.Errorf("deserialization worked on a wrong public key: %s", a)
	}
}

func TestStringerIsValid(t *testing.T) {
	pk1bytes := pkStringToBytes(t, publicKey1)
	if a, err := address.NewFromAddress("M00EXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4N3Qqa1", pk1bytes); err != nil {
		t.Errorf("problem with deserialization: %s", err)
	} else if a.String() != "M00EXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4N3Qqa1" {
		t.Errorf("stringer is incorrect: %s", testAE(a.String(), "M00EXMPnnaWFqRyVxWdhYCgGzpnaL4qBy4N3Qqa1"))
	}
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
			t.Errorf("invalid address passed validation: %s, test: %s", pair.invalidAddress, pair.testReason)
		}
	}
}
