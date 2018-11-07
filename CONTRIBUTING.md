*This is work in progress, it will turn into a Contributor's Guide in time.*

## Design Principles

This section explains our way of dealing with common concerns in the project.

### Go Context
TBD
* *When to use: Graceful shutdown, as Key-Value store?*
* *Contexts hierarchy*

### Timers
TBD
* *We modified the default Go implementation*

### Synchronization

Our coding conventions follow two alternative patterns for synchronization within a service:
* **Channel-based** - where a long living goroutine handles serially messages sent to it via a channel.
* **Mutex-based** - where a mutex is used for fine grained locks over shared data accessed by multiple goroutines.

#### Mutex best practices

* Use a `RWMutex` and separate between read locks and write locks. Do not assume that reading without a lock is safe.
* All state variables that are protected by the a mutex should be grouped in an anonymous struct that embeds the mutex.
* Unlocks of the mutex should be done using `defer mutex.Unlock()` and appear immediately after the locks. Function scope should be designed according to this principle to avoid locking the mutex for too long. It's best to create mutex access methods that lock the mutex, defer the unlock, perform the read/write and return.
* Locks should be for as short as possible (only while the data is accessed). Never make an out-of-service-bound call or an IO call when the mutex is locked.
* A mutex protects fields used together atomically. If two fields have different access patterns, they should be separated, each under its own mutex.

```golang
type inMemoryBlockPersistence struct {
	// this struct couples the data with a mutex that controls its access
	blockChain struct {
		sync.RWMutex
		blocks []*protocol.BlockPairContainer
	}
	
	failNextBlocks bool
	tracker        *synchronization.BlockTracker
	
	// this is another mutex-protected field, with different locking patterns
 	blockHeightsPerTxHash struct {
		sync.Mutex
		channels map[string]blockHeightChan
	}
}

func (bp *inMemoryBlockPersistence) addBlockToInMemoryChain(blockPair *protocol.BlockPairContainer) {
	bp.blockChain.Lock()
	defer bp.blockChain.Unlock()

	bp.blockChain.blocks = append(bp.blockChain.blocks, blockPair)
}

```
* Beware of the classic pitfall of (1) Read lock and unlock (2) Processing (3) Write lock and unlock. During phase (2) the read data might no longer be relevant due to another write. A way to mitigate this is to compare the read data during phase (3) to make sure it's still as expected and if not, abort the write.

### Error handling
The Orbs platform is a self-healing eco-system. This means that returning Go `Error`s is only meaningful as a logging tool.
Human intervention should not be required to fix a condition that caused an `Error`.
#### Unrecoverable errors
In the event of an unrecoverable error, the app panics and crashes.
This includes assertions on conditions that cannot happen unless there is a software bug, system errors such as **Out of memory** etc.

### Logging
TBD
* *Our logging framework*


### Monitoring
TBD
* Implementation in code
* Tools that display data

### Configuration


### Performance testing
TBD

### Testing Strategy
Contributions without full test coverage will _not_ be accepted. We use Test-Driven Development to help shape and evolve our design, and would prefer any contributed code to have been developed using TDD.

### CI
#### Downloading CircleCI log files
In case CircleCI does not display enough data on the build page (too much data, or CircleCI timed out before the test finished), you can still manually download its logs.
* Go to the specific build that you want to download logs for (see [example](https://circleci.com/gh/orbs-network/orbs-network-go/3328)), either from a PR's page, or [here](https://circleci.com/gh/orbs-network/orbs-network-go/tree/master)
* Click the 3rd tab from the left, "Artifacts"
* Expand the folder tree to uncover the list of log files
* If logs are archived in a tarball (bz2) file, download and expand it on your local machine
* If it's a long list of log files, you can use the Chrome Extension [Chrono Download Manager](https://chrome.google.com/webstore/detail/chrono-download-manager/mciiogijehkdemklbdcbfkefimifhecn?hl=en) to automatically download all logs
  * Install the extension, browse to the specific build's page (see [example](https://circleci.com/gh/orbs-network/orbs-network-go/3328))click its icon to the right of the URL pane, then click the magnifying glass icon.
  * Choose the files to download and click Download all.



