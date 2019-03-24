// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package leanhelixconsensus

import (
	"bytes"
	"encoding/binary"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	lhprotocol "github.com/orbs-network/lean-helix-go/spec/types/go/protocol"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
)

type keyManager struct {
	privateKey primitives.EcdsaSecp256K1PrivateKey
	logger     log.BasicLogger
}

// TODO Fix according to branch lh-outline, see https://tree.taiga.io/project/orbs-network/us/566

func NewKeyManager(logger log.BasicLogger, privateKey primitives.EcdsaSecp256K1PrivateKey) *keyManager {
	return &keyManager{
		logger:     logger,
		privateKey: privateKey,
	}
}

func (km *keyManager) SignConsensusMessage(blockHeight lhprimitives.BlockHeight, content []byte) lhprimitives.Signature {
	sig, _ := digest.SignAsNode(km.privateKey, content) // TODO(v1): handle error (log) https://tree.taiga.io/project/orbs-network/us/603
	return lhprimitives.Signature(sig)
}

func (km *keyManager) SignRandomSeed(blockHeight lhprimitives.BlockHeight, content []byte) lhprimitives.RandomSeedSignature {
	sig, _ := digest.SignAsNode(km.privateKey, content) // TODO(v1): handle error (log) https://tree.taiga.io/project/orbs-network/us/603
	return lhprimitives.RandomSeedSignature(sig)
}

func (km *keyManager) VerifyConsensusMessage(blockHeight lhprimitives.BlockHeight, content []byte, sender *lhprotocol.SenderSignature) error {
	return digest.VerifyNodeSignature(primitives.NodeAddress(sender.MemberId()), content, primitives.EcdsaSecp256K1Sig(sender.Signature()))
}

func (km *keyManager) VerifyRandomSeed(blockHeight lhprimitives.BlockHeight, content []byte, sender *lhprotocol.SenderSignature) error {

	// This is hack in v1 because BLS signatures / thresholds are not supported so master memberId is nil
	// See https://tree.taiga.io/project/orbs-network/us/565
	if len(sender.MemberId()) == 0 {
		sig := km.AggregateRandomSeed(blockHeight, nil)
		if bytes.Equal(sender.Signature(), sig) {
			return nil
		} else {
			return errors.Errorf("Mismatch in signature on blockHeight %s", blockHeight)
		}
		return nil
	}

	if err := digest.VerifyNodeSignature(primitives.NodeAddress(sender.MemberId()), content, primitives.EcdsaSecp256K1Sig(sender.Signature())); err != nil {
		return errors.Wrapf(err, "digest.VerifyNodeSignature() failed")
	}
	return nil
}

// This is hack in v1 - see https://tree.taiga.io/project/orbs-network/us/565
func (km *keyManager) AggregateRandomSeed(blockHeight lhprimitives.BlockHeight, randomSeedShares []*lhprotocol.SenderSignature) lhprimitives.RandomSeedSignature {
	heightAsByteArray := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightAsByteArray, uint64(blockHeight))
	return lhprimitives.RandomSeedSignature(hash.CalcSha256(heightAsByteArray))
}
