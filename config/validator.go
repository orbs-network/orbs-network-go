// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package config

import (
	"bytes"
	"encoding/hex"
	"github.com/orbs-network/crypto-lib-go/crypto/ethereum/digest"
	"github.com/orbs-network/crypto-lib-go/crypto/ethereum/signature"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
)

func ValidateNodeLogic(cfg NodeConfig) error {
	if MAXIMAL_PROTOCOL_VERSION_SUPPORTED_VALUE < MINIMAL_PROTOCOL_VERSION_SUPPORTED_VALUE {
		return errors.Errorf("Maximal Protocol version %d must be equal or higher than minimal Protocol Version %d", MAXIMAL_PROTOCOL_VERSION_SUPPORTED_VALUE, MINIMAL_PROTOCOL_VERSION_SUPPORTED_VALUE)
	}
	if cfg.BlockSyncNoCommitInterval() < cfg.BenchmarkConsensusRetryInterval() {
		return errors.Errorf("node sync timeout must be greater than benchmark consensus timeout (BlockSyncNoCommitInterval = %s, is greater than BenchmarkConsensusRetryInterval %s)",
			cfg.BlockSyncNoCommitInterval(), cfg.BenchmarkConsensusRetryInterval())
	}
	if cfg.BlockSyncNoCommitInterval() < cfg.LeanHelixConsensusRoundTimeoutInterval() {
		return errors.Errorf("node sync timeout must be greater than lean helix round timeout (BlockSyncNoCommitInterval = %s, is greater than LeanHelixConsensusRoundTimeoutInterval %s)",
			cfg.BlockSyncNoCommitInterval(), cfg.LeanHelixConsensusRoundTimeoutInterval())
	}
	if len(cfg.NodeAddress()) == 0 {
		return errors.New("node address must not be empty")
	}

	if cfg.SignerEndpoint() == "" {
		if len(cfg.NodePrivateKey()) == 0 {
			return errors.New("node private key must not be empty")
		}
		err := requireCorrectNodeAddressAndPrivateKey(cfg.NodeAddress(), cfg.NodePrivateKey())
		if err != nil {
			return err
		}
	}
	return nil
}

func requireCorrectNodeAddressAndPrivateKey(address primitives.NodeAddress, key primitives.EcdsaSecp256K1PrivateKey) error {
	msg := []byte{
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	}

	sign, err := signature.SignEcdsaSecp256K1(key, msg)
	if err != nil {
		return errors.Wrap(err, "could not create test sign")
	}

	recoveredPublicKey, err := signature.RecoverEcdsaSecp256K1(msg, sign)
	if err != nil {
		return errors.Wrap(err, "could not recover public key from test sign")
	}

	recoveredNodeAddress := digest.CalcNodeAddressFromPublicKey(recoveredPublicKey)
	if bytes.Compare(address, recoveredNodeAddress) != 0 {
		return errors.Errorf("node address %s derived from secret key does not match provided node address %s",
			hex.EncodeToString(recoveredNodeAddress), hex.EncodeToString(address))
	}
	return nil
}
