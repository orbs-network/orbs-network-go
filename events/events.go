package events


type Events interface {
	WaitForConsensusRounds(roundsToWait int)
	FinishedConsensusRound()

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
	e.consensusRoundsFinished <- true
}