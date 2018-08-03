package types

type Contract interface {
	// _init(ctx Context) error
}

type BaseContract struct {
	State StateSdk
}

func NewBaseContract(
	state StateSdk,
) *BaseContract {

	return &BaseContract{
		State: state,
	}
}
