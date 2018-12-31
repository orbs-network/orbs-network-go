package virtualmachine

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

func (s *service) getSignerAddress(signer *protocol.Signer) (primitives.ClientAddress, error) {
	switch signer.Scheme() {
	case protocol.SIGNER_SCHEME_EDDSA:
		return digest.CalcClientAddressOfEd25519Signer(signer)
	default:
		return nil, errors.New("transaction is not signed by any Signer")
	}
}
