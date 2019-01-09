package encoding

import (
	"encoding/hex"
	"strings"
)

func EncodeHex(data []byte) string {
	return "0x" + hex.EncodeToString(data)
}

// on decode error (eg. non hex character in str) returns zero_value, error
// on checksum failure returns decoded_value, error (so users could warn about checksum but still use the decoded)
func DecodeHex(str string) ([]byte, error) {
	if strings.HasPrefix(str, "0x") {
		str = str[2:]
	}
	return hex.DecodeString(str)
}
