package events

type Events interface {
	WaitForConsensusRounds(roundsToWait int)
	FinishedConsensusRound()
	ConsensusError(err error)

}

type events struct {
	consensusRoundsFinished chan bool
}

func NewEvents() Events {
	return &events{make(chan bool, 1000)}
}

func (e *events) WaitForConsensusRounds(roundsToWait int) {
	for i := 0; i < roundsToWait; i++ {
		<- e.consensusRoundsFinished
	}
}


func (e *events) FinishedConsensusRound() {
	//println("Finished consensus round")
	e.consensusRoundsFinished <- true
}

func (e *events) ConsensusError(err error)() {
	//println(fmt.Sprintf("Error during consensus: %s", err))
	e.consensusRoundsFinished <- true
}