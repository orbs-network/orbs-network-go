package blockstorage

import "github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"

type stateFn func(sync *blockSync) stateFn

// this is missing some channels, one for outputting blocks/chunks, maybe others for timeouts? probably missing more data that we need to keep states valid, need to implement first..
type blockSync struct {
	AvailabilityResponses []*gossipmessages.BlockAvailabilityResponseMessage
}

func sync() *blockSync {
	bs := &blockSync{
		AvailabilityResponses: nil,
	}

	go bs.mainLoop()
	return bs
}

// this may need to handle a more complex logic, although with the waitFor.. which is reacting for the 'waits' maybe this logic is good enough
func (sync *blockSync) mainLoop() {
	for state := noSyncRequired; state != nil; {
		state = state(sync)
	}

	// need to close channels if the loop terminates? although GC may handle this once block storage dies (if BS does not die we leak as it may reference still to this BS, so much BS in one place..)
}

// can return collecting or wait if not required to sync still
// is this really required? i think this will now only send us to 'waiting'
func noSyncRequired(sync *blockSync) stateFn {
	// should we sync?
	return nil
}

// this is a waiting logic state, it blocks, always, until something happens, it handles all the external events / timers
func waitForNextAction(sync *blockSync) stateFn {
	// the logic here is a select on channels, no default (so we do not busy wait)
	// 1. wait for the x sec timeout channel on no commits from storage - syncOnNoCommit, move to state collectingAvailability
	// 2. wait for the y sec timeout channel on timeout for the collecting avail - collectingAvailabilityFinished, move to state collectingAvailability (or should we have a state called finishedCollectingAvailability, makes more sense to separate the logic)
	// 3. wait for the z sec timeout channel on timeout for waiting for chunk - chunkNeverArrived, move to state collecting Availability, we need to send a new request

	// all of these channels should be a empty struct channels, just notifications / events on behavior
	// timer management is required to keep this state valid, once a chunk arrives, timer #3 should be destroyed so it does not ping (probably drained and stopped? so we can use our timer implementation that handles that)
	// timer #2 will always fire (once)
	// timer #1 is managed in block storage, and should fire 'every 8 seconds' assuming we are in a vacuum and no commits will ever arrive

	// i think that all timer management happens outside this stage, which is impossible for #3, as we will block in this state, maybe we should think of a different way to handle it then, maybe add another waiting channel on 'chunkArrived', and then 'processingChunk' makes a lot of sense (see below)
	return nil
}

// can return idle or waiting or collecting chunk
func collectingAvailability(sync *blockSync) stateFn {
	// maybe we should split this to 'collecting' and 'finishedCollecting' (see waitFor..)
	return nil
}

// can return collecting avail or collecting chunk (self, timeout pending)
func collectingChunk(sync *blockSync) stateFn {
	// possibly, there may be a new state, 'processingChunk' but need to 'feel' it once integration with block storage is done, makes more sense to send it out via a channel
	return nil
}
