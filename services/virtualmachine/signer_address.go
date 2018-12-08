package virtualmachine

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

// TODO(v1): maybe move all of these actions (the hashes) to crypto/digest since the client needs them as well

func (s *service) getSignerAddress(signer *protocol.Signer) (primitives.Ripmd160Sha256, error) {
	switch signer.Scheme() {
	case protocol.SIGNER_SCHEME_EDDSA:
		return addressEd25519Signer(signer)
	default:
		return nil, errors.New("transaction is not signed by any Signer")
	}
}

func addressEd25519Signer(signer *protocol.Signer) (primitives.Ripmd160Sha256, error) {
	signerPublicKey := signer.Eddsa().SignerPublicKey()
	if len(signerPublicKey) != keys.ED25519_PUBLIC_KEY_SIZE_BYTES {
		return nil, errors.New("transaction is not signed by a valid Signer")
	}
	return hash.CalcRipmd160Sha256(signerPublicKey), nil
}

// TODO(v1): add argument (spec feature)
func addressContractCall(contractName primitives.ContractName) (primitives.Ripmd160Sha256, error) {
	if len(contractName) == 0 {
		return nil, errors.New("contract name is missing for addressing")
	}
	return hash.CalcRipmd160Sha256([]byte(contractName)), nil
}
