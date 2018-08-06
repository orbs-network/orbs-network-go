package types

// Contract receiver for repository contracts (instantiated only on repository init = system init)
// TODO: consider merging the Contract receiver into the contract Context so we don't have two separate mechanisms

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
