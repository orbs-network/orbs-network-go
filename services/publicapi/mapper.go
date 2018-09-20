package publicapi

import (
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	)

//import "context"

type resultWaiter chan *services.AddNewTransactionOutput


type mapper struct {
	m map[string]resultWaiter
}

func newMapper(/*ctx context.Context*/) *mapper {
	// TODO supervise
	result := &mapper{m: map[string]resultWaiter{}}
	return result
}

func (m *mapper) addWaiter(k string) (resultWaiter, error) {
	if _, exists := m.m[k]; !exists {
		m.m[k] = make(resultWaiter)
		return m.m[k], nil
	}
	return nil, errors.New("Already exists")
}

func (m *mapper) delete(k string) (resultWaiter, error) {
	if wc, exists := m.m[k]; !exists {
		return nil, errors.New("doesn't exists")
	} else {
		delete(m.m, k)
		return wc, nil
	}
}
