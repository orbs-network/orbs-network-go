// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package adapter

import "time"

const SOURCE_CODE_PATH = "native-src"
const SHARED_OBJECT_PATH = "native-bin"
const GC_CACHE_PATH = "native-cache"
const MAX_COMPILATION_TIME = 10 * time.Second
// in a poor CPU environment when we have many containers starting up
// in the same time (usually on our CI) or when running Docker e2e locally
// We almost always end up with CPU starvation when the warm up compilation occurs
//  (Due to all containers executing a "go build ..." shell at the same time.
// Setting a higher time simply solves this issue and prevent a lot of side effects
// from happening.
const MAX_WARM_UP_COMPILATION_TIME = 45 * time.Second

type Config interface {
	ProcessorArtifactPath() string
	ProcessorPerformWarmUpCompilation() bool
}
