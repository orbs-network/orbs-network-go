// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package leanhelixconsensus

import (
	"bytes"
	"context"
	"encoding/binary"
	"github.com/orbs-network/crypto-lib-go/crypto/digest"
	"github.com/orbs-network/crypto-lib-go/crypto/hash"
	"github.com/orbs-network/crypto-lib-go/crypto/signer"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	lhprotocol "github.com/orbs-network/lean-helix-go/spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
)

type keyManager struct {
	signer signer.Signer
	logger log.Logger
}

// TODO Fix according to branch lh-outline, see https://tree.taiga.io/project/orbs-network/us/566

func NewKeyManager(logger log.Logger, signer signer.Signer) *keyManager {
	return &keyManager{
		logger: logger,
		signer: signer,
	}
}

func (km *keyManager) SignConsensusMessage(ctx context.Context, blockHeight lhprimitives.BlockHeight, content []byte) lhprimitives.Signature {
	sig, err := km.signer.Sign(ctx, content) // TODO(v1): handle error (log) https://tree.taiga.io/project/orbs-network/us/603
	if err != nil {
		km.logger.Error("failed to sign consensus message", log.Error(err))
	}
	return lhprimitives.Signature(sig)
}

func (km *keyManager) SignRandomSeed(ctx context.Context, blockHeight lhprimitives.BlockHeight, content []byte) lhprimitives.RandomSeedSignature {
	sig, err := km.signer.Sign(ctx, content) // TODO(v1): handle error (log) https://tree.taiga.io/project/orbs-network/us/603
	if err != nil {
		km.logger.Error("failed to sign random seed", log.Error(err))
	}
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
