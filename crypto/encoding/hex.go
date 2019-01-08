package encoding

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"strings"
)

func EncodeHex(data []byte) string {
	result := []byte(hex.EncodeToString(data))
	hashed := hash.CalcSha256(data)

	for i := 0; i < len(result); i++ {
		hashByte := hashed[i/2]
		if i%2 == 0 {
			hashByte = hashByte >> 4
		} else {
			hashByte &= 0xf
		}

		if result[i] > '9' && hashByte > 7 {
			result[i] -= 32
		}
	}

	return "0x" + string(result)
}

// on decode error (eg. non hex character in str) returns zero_value, error
// on checksum failure returns decoded_value, error (so users could warn about checksum but still use the decoded)
// if all is lower or upper then the check is ignored (as the checksum was probably not taken into account)
func DecodeHex(str string) ([]byte, error) {
	if strings.HasPrefix(str, "0x") {
		str = str[2:]
	}

	return hex.DecodeString(str)
}
