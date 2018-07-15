package address

import (
	"crypto/sha256"
	"golang.org/x/crypto/ripemd160"
	"fmt"
	"hash/crc32"
)

type Address struct {
	networkId string
	virtualChainId string
	publicKey []byte
	version int
	accountId []byte
	checksum uint32
	fullAddress []byte
}

const (
	MAIN_NETWORK_ID  = 'M'
	TEST_NETWORK_ID  = 'T'
	SYSTEM_VCHAIN_ID = "000000"
)

func (a Address) CreateFromPK(publicKey []byte, virtualChainId string, networkId string) (Address, error) {
	newAddress := Address{
		publicKey:      publicKey,
		virtualChainId: virtualChainId,
		networkId:      networkId,
	}
	_, err := newAddress.createAccountId()
	if err != nil {
		return Address{}, err
	}

	return newAddress, nil
}

func (a *Address) VirtualChainId() string {
	return a.virtualChainId
}

func (a *Address) NetworkId() string {
	return a.networkId
}

func (a *Address) Version() int {
	return 0
}

func (a *Address) createAccountId() (bool, error) {
	if a.publicKey == nil || len(a.publicKey) == 0 {
		return false, fmt.Errorf("public key invalid, cannot create account id. pk: %#v", a.publicKey)
	}

	sha256digest := sha256.Sum256(a.publicKey)
	r := ripemd160.New()
	r.Write(sha256digest[:])
	a.accountId = r.Sum(nil)
	return true, nil
}

func (a *Address) AccountId() []byte {
	return a.accountId
}

func (a *Address) calculateChecksum() (bool, error) {
	if a.fullAddress == nil {
		return false, fmt.Errorf("full address not available yet, cannot calculate checksum")
	}
	a.checksum = crc32.ChecksumIEEE(a.fullAddress)
	return true, nil
}