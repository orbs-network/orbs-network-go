package adapter

import "time"

const SOURCE_CODE_PATH = "native-src"
const SHARED_OBJECT_PATH = "native-bin"
const GC_CACHE_PATH = "native-cache"
const MAX_COMPILATION_TIME = 10 * time.Second
const MAX_WARM_UP_COMPILATION_TIME = 15 * time.Second

type Config interface {
	ProcessorArtifactPath() string
}
