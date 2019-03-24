// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package keys

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
)

type ed25519KeyPairHex struct {
	publicKey  string
	privateKey string
}

var ed25519KeyPairs = []ed25519KeyPairHex{
	{"dfc06c5be24a67adee80b35ab4f147bb1a35c55ff85eda69f40ef827bddec173", "93e919986a22477fda016789cca30cb841a135650938714f85f0000a65076bd4dfc06c5be24a67adee80b35ab4f147bb1a35c55ff85eda69f40ef827bddec173"},
	{"92d469d7c004cc0b24a192d9457836bf38effa27536627ef60718b00b0f33152", "3b24b5f9e6b1371c3b5de2e402a96930eeafe52111bb4a1b003e5ecad3fab53892d469d7c004cc0b24a192d9457836bf38effa27536627ef60718b00b0f33152"},
	{"a899b318e65915aa2de02841eeb72fe51fddad96014b73800ca788a547f8cce0", "2c72df84be2b994c32a3f4ded0eab901debd3f3e13721a59eed00fbd1da4cc00a899b318e65915aa2de02841eeb72fe51fddad96014b73800ca788a547f8cce0"},
	{"58e7ed8169a151602b1349c990c84ca2fb2f62eb17378f9a94e49552fbafb9d8", "163987afcee69969cae3528161d84e32f76b09bbf0dd77dd704e5cb915c7d56f58e7ed8169a151602b1349c990c84ca2fb2f62eb17378f9a94e49552fbafb9d8"},
	{"23f97918acf48728d3f25a39a5f091a1a9574c52ccb20b9bad81306bd2af4631", "74b63e4f6f908ac42c1b4c7b3b6028c7b665df4375c1acbf4dce2b1b91aefc5b23f97918acf48728d3f25a39a5f091a1a9574c52ccb20b9bad81306bd2af4631"},
	{"07492c6612f78a47d7b6a18a17792a01917dec7497bdac1a35c477fbccc3303b", "d9fae84f80b842f57770a9ae67c7eb58ce502eb32502d43ddec5da115ccd2e2107492c6612f78a47d7b6a18a17792a01917dec7497bdac1a35c477fbccc3303b"},
	{"43a4dbbf7a672c6689dbdd662fd89a675214b00d884bb7113d3410b502ecd826", "c7c2579fb128bf1d687081600f171060d95da22543920ea3490d8e71980babe943a4dbbf7a672c6689dbdd662fd89a675214b00d884bb7113d3410b502ecd826"},
	{"469bd276271aa6d59e387018cf76bd00f55c702931c13e80896eec8a32b22082", "0d953392b90e5cf5f0162cb289ff1b77a358921201aa5c91c902b38aa22a1878469bd276271aa6d59e387018cf76bd00f55c702931c13e80896eec8a32b22082"},
	{"102073b28749be1e3daf5e5947605ec7d43c3183edb48a3aac4c9542cdbaf748", "57249e0b586083a60df94044971416cb9fdd373855aac9e04bceb4c96e53559e102073b28749be1e3daf5e5947605ec7d43c3183edb48a3aac4c9542cdbaf748"},
	{"70d92324eb8d24b7c7ed646e1996f94dcd52934a031935b9ac2d0e5bbcfa357c", "f1c41ba8a1d78f7cdc4f4ff23f3b736e30c630085697d6503e16ac899646f5ab70d92324eb8d24b7c7ed646e1996f94dcd52934a031935b9ac2d0e5bbcfa357c"},
}

func Ed25519KeyPairForTests(setIndex int) *keys.Ed25519KeyPair {
	if setIndex > len(ed25519KeyPairs) {
		return nil
	}

	pub, err := hex.DecodeString(ed25519KeyPairs[setIndex].publicKey)
	if err != nil {
		return nil
	}

	pri, err := hex.DecodeString(ed25519KeyPairs[setIndex].privateKey)
	if err != nil {
		return nil
	}

	return keys.NewEd25519KeyPair(pub, pri)
}
