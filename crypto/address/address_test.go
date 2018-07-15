package address_test

import (
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

func TestAddressInitializationWithPublicKeyOnTestNet(t *testing.T) {
	t.Error("need to implement")
}

func TestAddressInitializationWithKeyOnMainNet(t *testing.T) {
	t.Error("need to implement")
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
	//for _, test := range validAddressTests {
	//	t.Error("need to implement")
	//}
}

func TestAddressIsValidFails(t *testing.T) {
	//for _, pair := range invalidAddressTests {
	//	t.Error("need to implement")
	//}
}
