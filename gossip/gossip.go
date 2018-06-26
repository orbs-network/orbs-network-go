package gossip

type Gossip interface {
	RegisterAll(listeners []Listener)
	ForwardTransaction(value int)
}

type gossip struct {
	listeners []Listener
}

func NewGossip() Gossip {
	return &gossip{}
}

func (g *gossip) RegisterAll(listeners []Listener) {
	g.listeners = listeners
}

func (g *gossip) ForwardTransaction(value int) {
	for _, l := range g.listeners {
		l.OnForwardedTransaction(value)
	}
}