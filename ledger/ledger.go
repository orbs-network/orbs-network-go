package ledger

type Ledger interface {
	AddTransaction(value int)
	GetState() int
}

type ledger struct {
	values []int
}

func NewLedger() Ledger {
	return &ledger{}
}

func (l *ledger) AddTransaction(value int) {
	l.values = append(l.values, value)
}

func (l *ledger) GetState() int {
	sum := 0
	for _, num := range l.values {
		sum += num
	}
	return sum

}
