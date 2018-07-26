package gossip

import (
	"encoding/hex"
	"fmt"
)

type ErrCorruptHeader struct {
	RawHeader []byte
}

func (e *ErrCorruptHeader) Error() string {
	return fmt.Sprintf("gossip header cannot be parsed: %v", hex.EncodeToString(e.RawHeader))
}

type ErrCodecEncode struct {
	Type   string
	Object interface{}
}

func (e *ErrCodecEncode) Error() string {
	return fmt.Sprintf("gossip codec cannot encode %s: %v", e.Type, e.Object)
}

type ErrCodecDecode struct {
	Type     string
	Payloads [][]byte
}

func (e *ErrCodecDecode) Error() string {
	hexPayloads := []string{}
	for _, payload := range e.Payloads {
		hexPayloads = append(hexPayloads, hex.EncodeToString(payload))
	}
	return fmt.Sprintf("gossip codec cannot decode %s: %v", e.Type, hexPayloads)
}
