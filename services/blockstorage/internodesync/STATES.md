# Block Sync State Machine

This describes the block sync state machine implementation and flow, according to [the spec](https://github.com/orbs-network/orbs-spec/blob/master/behaviors/services/block-storage.md).

## States

* Idle
* Collecting Availability Responses (collecting or car)
* Finished Collecting Availability Responses (finishedCollecting or fcar)
* Waiting For Chunks (waiting)
* Processing Blocks (processing)

## Timers

* Idle state timeout, triggers when we receive no blocks for X seconds
* Collecting state timeout - always defined and awaits for responses to arrive
* Waiting state timeout - happens when the source selected to sync does not send us the responses until this timeout expires

## State Transition Logic

### Init
The system starts at the collecting state

### Idle Flow
When the system is idle and not syncing, it will be in idle state.

> idle -> idle

Idle can transition to idle if a new block notification arrives from gossip
(from the consensus most likely)

> idle -> collecting

Idle can transition to collecting if the timeout expires waiting to new blocks, this is defined in the spec and is 8 seconds as of writing this

### Collecting Availability Responses Flow
Collecting is the broadcast stage where we are looking for peers

> collecting -> finished collecting

Collecting will transition to finished collecting, always, after the waiting period for responses expires

### Finished Collecting Availability Responses
Finished collecting is just a mediator which decides if we can begin sync with some source server or not

> finished collecting -> idle

Finished collecting will transition to idle in the event where we received not responses from the collecting availability responses state

> finished collecting -> waiting

Finished collecting will transition to waiting when responses have arrived and a source was chosen to be the sync peer

### Waiting for Chunks Flow
Waiting for chunks is when we are broadcasting to the source our request for chunks and are waiting for the blocks to be sent

> waiting -> idle

We jump back to idle when the timeout for waiting for the chunks has expired

> waiting -> processing

Waiting will transition to processing when the blocks are received from the source

### Processing Blocks Flow
Processing blocks is where we commit the blocks received from sync

> processing -> idle

Processing will transition to idle where there is an exception flow, for example, the blocks never arrived

> processing -> collecting

When we finished committing all blocks received, we return to collecting state (as there may be more data we want)
