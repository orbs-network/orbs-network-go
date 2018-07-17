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
	checksum       *uint32
	fullAddress    []byte
}

const (
	MAIN_NETWORK_ID      = "M"
	TEST_NETWORK_ID      = "T"
	ADDRESS_LENGTH       = 40
	VIRTUAL_CHAIN_ID_MSB = 0X08
	VCHAIN_ID_SIZE       = 3
	NETWORK_ID_SIZE      = 1
	VERSION_SIZE         = 1
	ACCOUNT_ID_SIZE      = 20
	CHECKSUM_SIZE        = 4
)

func NewFromPK(publicKey []byte, virtualChainId string, networkId string) (Address, error) {
	newAddress := Address{
		publicKey:      publicKey,
		virtualChainId: virtualChainId,
		networkId:      networkId,
		version:        0,
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

func calculateAccountId(publicKey []byte) ([]byte, error) {
	if len(publicKey) == 0 {
		return nil, fmt.Errorf("public key invalid, cannot create account id")
	}

	sha256digest := sha256.Sum256(publicKey)
	r := ripemd160.New()
	_, err := r.Write(sha256digest[:])
	if err != nil {
		return nil, err
	}
	accountId := r.Sum(nil)
	return accountId, nil
}

func (a *Address) AccountId() ([]byte, error) {
	if a.accountId == nil {
		if result, err := calculateAccountId(a.publicKey); err != nil {
			return nil, err
		} else {
			a.accountId = result
		}
	}
	return a.accountId, nil
}

func calculateChecksum(fullAddress []byte) (*uint32, error) {
	if len(fullAddress) == 0 {
		return nil, fmt.Errorf("full address not available, cannot calculate checksum")
	}
	checksum := crc32.ChecksumIEEE(fullAddress)
	return &checksum, nil
}

func (a *Address) Checksum() (uint32, error) {
	if a.checksum == nil {
		if result, err := calculateChecksum(a.fullAddress); err != nil {
			return 0, err
		} else {
			a.checksum = result
		}
	}
	return *a.checksum, nil
}

func (a *Address) Raw() ([]byte, error) {
	if fullAddress, err := a.generateFullAddress(); err != nil {
		return nil, err
	} else if checksum, err := a.Checksum(); err != nil {
		return nil, err
	} else {
		csBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(csBytes, checksum)
		return append(fullAddress, csBytes...), nil
	}
}

func ToBase58(rawAddress []byte) string {
	bs58 := fmt.Sprintf("%s%s%s", rawAddress[:1], hex.EncodeToString(rawAddress[1:2]), base58.Encode(rawAddress[2:]))
	return bs58
}

func (a *Address) generateFullAddress() ([]byte, error) {
	if a.fullAddress == nil {
		// scheme is nID|v|vchID|aID
		if aID, err := a.AccountId(); err != nil {
			return nil, err
		} else {
			a.fullAddress, err = generateFullAddress(a.networkId, a.version, a.virtualChainId, aID)
			if err != nil {
				return nil, err
			}
		}
	}
	return a.fullAddress, nil
}

func generateFullAddress(networkId string, version uint8, vchainId string, accountId []byte) ([]byte, error) {
	fa := make([]byte, NETWORK_ID_SIZE+VERSION_SIZE+VCHAIN_ID_SIZE)
	fa[0] = byte(networkId[0])
	fa[1] = byte(version)
	if vchainBytes, err := hex.DecodeString(vchainId); err != nil {
		return nil, err
	} else {
		for i, vcb := range vchainBytes {
			fa[i+NETWORK_ID_SIZE+VERSION_SIZE] = vcb
		}
	}
	fa = append(fa, accountId...)
	return fa, nil
}

func IsValid(address string) (bool, error) {
	if len(address) != ADDRESS_LENGTH {
		return false, fmt.Errorf("length mismatch")
	}

	net := string(address[0])
	if net != MAIN_NETWORK_ID && net != TEST_NETWORK_ID {
		return false, fmt.Errorf("network id invalid")
	}

	version, err := hex.DecodeString(string(address[1:3]))
	if err != nil {
		return false, fmt.Errorf("version parsing failed, %s", err)
	}
	if version[0] != 0 {
		return false, fmt.Errorf("invalid version")
	}

	decoded, err := base58.Decode([]byte(address[3:]))
	if err != nil {
		return false, fmt.Errorf("base58 decode failed: %s", err)
	}

	// decoded: 3-vchain|20-account|4-checksum
	if len(decoded) != VCHAIN_ID_SIZE+ACCOUNT_ID_SIZE+CHECKSUM_SIZE {
		return false, fmt.Errorf("decoded part invalid")
	}

	vchainID := decoded[0:VCHAIN_ID_SIZE]
	if vchainID[0] < VIRTUAL_CHAIN_ID_MSB {
		return false, fmt.Errorf("vchain id invalid")
	}

	accountId := decoded[VCHAIN_ID_SIZE : ACCOUNT_ID_SIZE+VCHAIN_ID_SIZE]
	expectedCs := binary.BigEndian.Uint32(decoded[ACCOUNT_ID_SIZE+VCHAIN_ID_SIZE:])
	if fa, err := generateFullAddress(net, version[0], hex.EncodeToString(vchainID), accountId); err != nil {
		return false, fmt.Errorf("failed to generate full address: %s", err)
	} else if actualCs, err := calculateChecksum(fa); err != nil {
		return false, fmt.Errorf("failed to calculate checksum: %s", err)
	} else {
		if !(expectedCs == *actualCs) {
			return false, fmt.Errorf("checksum does not match address")
		}
	}

	return true, nil
}
