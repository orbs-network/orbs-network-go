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
