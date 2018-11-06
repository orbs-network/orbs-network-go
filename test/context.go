package test

import (
	"context"
)

func WithContext(f func(ctx context.Context)) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f(ctx)
}
