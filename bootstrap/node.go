package bootstrap

type Node interface {
	SendTransaction(value int) (int, error)
	CallMethod() (int, error)
}

type node struct {
	value int
}

func NewNode() Node {
	return &node{}
}

func (n *node) SendTransaction(value int) (int, error) {
	n.value = value
	return n.value, nil
}

func (n *node) CallMethod() (int, error) {
	return n.value, nil
}
