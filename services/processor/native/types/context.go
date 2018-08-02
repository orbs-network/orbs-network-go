package types

type ContractContext interface {
}

type BaseContext struct {
}

func NewBaseContext() *BaseContext {
	return &BaseContext{}
}
