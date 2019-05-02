// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package config

// these get written in build-binaries.sh during linking
var SemanticVersion string
var CommitVersion string

type Version struct {
	Semantic string
	Commit   string
}

func GetVersion() Version {
	return Version{
		Semantic: SemanticVersion,
		Commit:   CommitVersion,
	}
}

func (v Version) String() string {
	return v.Semantic + "\n" + v.Commit
}
