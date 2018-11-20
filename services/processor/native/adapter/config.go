package adapter

import "time"

const SOURCE_CODE_PATH = "native-src"
const SHARED_OBJECT_PATH = "native-bin"
const GC_CACHE_PATH = "native-cache"
const MAX_COMPILATION_TIME = 5 * time.Second          // TODO: maybe move to config or maybe have caller provide via context
const MAX_WARM_UP_COMPILATION_TIME = 15 * time.Second // TODO: maybe move to config or maybe have caller provide via context

type Config interface {
	ProcessorArtifactPath() string
}

