package address

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/base58"
	"golang.org/x/crypto/ripemd160"
	"hash/crc32"
)

type Address struct {
	networkId      string
	virtualChainId string
	publicKey      []byte
	version        uint8
	accountId      []byte
	checksum       uint32
	fullAddress    []byte
}

const (
	MAIN_NETWORK_ID  = "M"
	TEST_NETWORK_ID  = "T"
	SYSTEM_VCHAIN_ID = "000000"
)

func CreateFromPK(publicKey []byte, virtualChainId string, networkId string) (Address, error) {
	newAddress := Address{
		publicKey:      publicKey,
		virtualChainId: virtualChainId,
		networkId:      networkId,
		version:        0,
	}
	_, err := newAddress.createAccountId()
	if err != nil {
		return Address{}, err
	}

	_, err = newAddress.generateFullAddress()
	if err != nil {
		return Address{}, err
	}

	_, err = newAddress.calculateChecksum()
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

func (a *Address) Version() uint8 {
	return a.version
}

func (a *Address) createAccountId() (bool, error) {
	if a == nil || len(a.publicKey) == 0 {
		return false, fmt.Errorf("public key invalid, cannot create account id")
	}

	sha256digest := sha256.Sum256(a.publicKey)
	r := ripemd160.New()
	_, err := r.Write(sha256digest[:])
	if err != nil {
		return false, err
	}
	a.accountId = r.Sum(nil)
	return true, nil
}

func (a *Address) AccountId() []byte {
	return a.accountId
}

func (a *Address) Checksum() uint32 {
	return a.checksum
}

func (a *Address) RawAddress() []byte {
	cs := make([]byte, 4)
	binary.BigEndian.PutUint32(cs, a.checksum)
	return append(a.fullAddress, cs...)
}

func ToBase58(rawAddress []byte) string {
	bs58 := fmt.Sprintf("%s%s%s", rawAddress[:1], hex.EncodeToString(rawAddress[1:2]), base58.Encode(rawAddress[2:]))
	return bs58
}

func (a *Address) calculateChecksum() (bool, error) {
	if a.fullAddress == nil || len(a.fullAddress) == 0 {
		return false, fmt.Errorf("full address not available yet, cannot calculate checksum")
	}
	a.checksum = crc32.ChecksumIEEE(a.fullAddress)
	return true, nil
}

func (a *Address) generateFullAddress() (bool, error) {
	networkPart := []byte(a.networkId)
	versionPart := make([]byte, 1)
	versionPart[0] = byte(a.version)
	vchainIdPart, err := hex.DecodeString(a.virtualChainId)
	if err != nil {
		return false, err
	}
	a.fullAddress = append(networkPart, versionPart...)
	a.fullAddress = append(a.fullAddress, vchainIdPart...)
	a.fullAddress = append(a.fullAddress, a.accountId...)
	return true, nil
}
