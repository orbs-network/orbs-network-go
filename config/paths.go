// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package config

import (
	"path/filepath"
	"runtime"
)

func GetProjectSourceRootPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..")
}

func GetCurrentSourceFileDirPath() string {
	_, filename, _, _ := runtime.Caller(1)
	return filepath.Dir(filename)
}

func GetProjectSourceTmpPath() string {
	return filepath.Join(GetProjectSourceRootPath(), "_tmp")
}
