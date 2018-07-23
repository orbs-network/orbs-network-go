package address

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/base58"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ripemd160"
	"hash/crc32"
)

type Address struct {
	networkId      string
	virtualChainId []byte
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
	PUBLIC_KEY_SIZE      = 32
)

func validateAddressParts(pk []byte, vcId []byte, net string, v uint8) error {
	if len(pk) != PUBLIC_KEY_SIZE {
		return fmt.Errorf("pk invalid (%v), cannot create address", pk)
	}

	if !IsValidVChainId(vcId) {
		return fmt.Errorf("invalid virtual chain id (%v), cannot create address", vcId)
	}

	if !IsValidNetworkId(net) {
		return fmt.Errorf("invalid network id (%s), cannot create address", net)
	}

	if !IsValidVersion(v) {
		return fmt.Errorf("invalid version (%d), cannot create address", v)
	}

	return nil
}

func NewFromPK(publicKey []byte, virtualChainId string, networkId string) (*Address, error) {
	vcidBytes, err := hex.DecodeString(virtualChainId)
	if err != nil {
		return nil, fmt.Errorf("unable to convert vcid from hex to bytes (%s)", virtualChainId)
	}

	if err := validateAddressParts(publicKey, vcidBytes, networkId, 0); err != nil {
		return nil, err
	}

	newAddress := Address{
		publicKey:      publicKey,
		virtualChainId: vcidBytes,
		networkId:      networkId,
		version:        0,
	}

	return &newAddress, nil
}

func NewFromAddress(a string, publicKey []byte) (*Address, error) {
	if _, err := IsValid(a); err != nil {
		return nil, errors.Wrap(err, "address is invalid")
	}

	if raw, err := Base58Decode(a); err != nil {
		return nil, errors.Wrap(err, "address is invalid")
	} else {
		return NewFromRaw(raw, publicKey)
	}
}

func NewFromRaw(a []byte, publicKey []byte) (*Address, error) {
	networkId := string(a[0])
	version := a[VERSION_SIZE]
	vcidBytes := a[NETWORK_ID_SIZE+VERSION_SIZE : NETWORK_ID_SIZE+VERSION_SIZE+VCHAIN_ID_SIZE]
	accountId := a[NETWORK_ID_SIZE+VERSION_SIZE+VCHAIN_ID_SIZE : NETWORK_ID_SIZE+VERSION_SIZE+VCHAIN_ID_SIZE+ACCOUNT_ID_SIZE]

	if err := validateAddressParts(publicKey, vcidBytes, networkId, version); err != nil {
		return nil, errors.Wrap(err, "failed to validate raw address: %s")
	}

	newAddress := Address{
		publicKey:      publicKey,
		virtualChainId: vcidBytes,
		networkId:      networkId,
		version:        0,
	}

	if aid, err := newAddress.AccountId(); err != nil {
		return nil, errors.Wrap(err, "unable to create account id for new address")
	} else if !bytes.Equal(aid, accountId) {
		return nil, fmt.Errorf("account id mismatch, pk does not match invalid address")
	}

	cs := binary.BigEndian.Uint32(a[NETWORK_ID_SIZE+VERSION_SIZE+VCHAIN_ID_SIZE+ACCOUNT_ID_SIZE:])
	if _, err := newAddress.generateFullAddress(); err != nil {
		return nil, errors.Wrap(err, "failed to generate full address for new address during checksum test")
	}

	if actualCs, err := newAddress.Checksum(); err != nil {
		return nil, errors.Wrap(err, "failed to generate full address for new address during checksum test")
	} else if cs != actualCs {
		return nil, fmt.Errorf("checksum mismatch, cannot create address")
	}

	return &newAddress, nil
}

func (a *Address) VirtualChainId() string {
	vcidStr := hex.EncodeToString(a.virtualChainId)
	return vcidStr
}

func (a *Address) NetworkId() string {
	return a.networkId
}

func (a *Address) Version() uint8 {
	return a.version
}

func IsValidVersion(v uint8) bool {
	return v == 0
}

func IsValidNetworkId(id string) bool {
	return id == MAIN_NETWORK_ID || id == TEST_NETWORK_ID
}

func IsValidVChainId(vchain []byte) bool {
	return len(vchain) == VCHAIN_ID_SIZE && vchain[0] >= VIRTUAL_CHAIN_ID_MSB
}

func calculateAccountId(publicKey []byte) ([]byte, error) {
	if len(publicKey) != PUBLIC_KEY_SIZE {
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
			return nil, errors.Wrap(err, "failed to create account id")
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
			return 0, errors.Wrap(err, "failed to create checksum")
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

func (a *Address) String() string {
	if r, err := a.Raw(); err == nil {
		return Base58Encode(r)
	}
	return "address object invalid"
}

func Base58Encode(rawAddress []byte) string {
	bs58 := fmt.Sprintf("%s%s%s", rawAddress[:1], hex.EncodeToString(rawAddress[1:2]), base58.Encode(rawAddress[2:]))
	return bs58
}

func Base58Decode(address string) ([]byte, error) {
	net := address[0]
	version, err := hex.DecodeString(address[1:3])
	if err != nil {
		return nil, errors.Wrap(err, "failed in decode version part")
	}
	fa, err := base58.Decode([]byte(address[3:]))
	if err != nil {
		return nil, errors.Wrap(err, "failed in 'fullAddress' decode part")
	}

	raw := make([]byte, NETWORK_ID_SIZE)
	raw[0] = net
	raw = append(raw, version...)
	raw = append(raw, fa...)

	return raw, nil
}

func (a *Address) generateFullAddress() ([]byte, error) {
	if a.fullAddress == nil {
		// scheme is nID|v|vchID|aID
		if aID, err := a.AccountId(); err != nil {
			return nil, err
		} else {
			a.fullAddress = generateFullAddress(a.networkId, a.version, a.virtualChainId, aID)
		}
	}
	return a.fullAddress, nil
}

func generateFullAddress(networkId string, version uint8, vchainId []byte, accountId []byte) []byte {
	fa := make([]byte, NETWORK_ID_SIZE+VERSION_SIZE+VCHAIN_ID_SIZE)
	fa[0] = byte(networkId[0])
	fa[1] = byte(version)
	for i, vcb := range vchainId {
		fa[i+NETWORK_ID_SIZE+VERSION_SIZE] = vcb
	}
	fa = append(fa, accountId...)
	return fa
}

func IsValid(address string) (bool, error) {
	if len(address) != ADDRESS_LENGTH {
		return false, fmt.Errorf("length mismatch")
	}

	net := string(address[0])
	if !IsValidNetworkId(net) {
		return false, fmt.Errorf("network id invalid")
	}

	version, err := hex.DecodeString(string(address[1:3]))
	if err != nil {
		return false, errors.Wrap(err, "version parsing failed")
	}
	if !IsValidVersion(version[0]) {
		return false, fmt.Errorf("invalid version")
	}

	decoded, err := base58.Decode([]byte(address[3:]))
	if err != nil {
		return false, errors.Wrap(err, "base58 decode failed")
	}

	// decoded: 3-vchain|20-account|4-checksum
	if len(decoded) != VCHAIN_ID_SIZE+ACCOUNT_ID_SIZE+CHECKSUM_SIZE {
		return false, fmt.Errorf("decoded part invalid")
	}

	vchainID := decoded[0:VCHAIN_ID_SIZE]
	if !IsValidVChainId(vchainID) {
		return false, fmt.Errorf("vchain id invalid")
	}

	accountId := decoded[VCHAIN_ID_SIZE : ACCOUNT_ID_SIZE+VCHAIN_ID_SIZE]
	expectedCs := binary.BigEndian.Uint32(decoded[ACCOUNT_ID_SIZE+VCHAIN_ID_SIZE:])
	fa := generateFullAddress(net, version[0], vchainID, accountId)
	if actualCs, err := calculateChecksum(fa); err != nil {
		return false, errors.Wrap(err, "failed to calculate checksum")
	} else {
		if expectedCs != *actualCs {
			return false, fmt.Errorf("checksum does not match address")
		}
	}

	return true, nil
}
