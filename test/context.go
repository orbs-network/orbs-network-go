package test

import (
	"context"
	"time"
)

func WithContext(f func(ctx context.Context)) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f(ctx)
}

func WithContextWithTimeout(d time.Duration, f func(ctx context.Context)) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	f(ctx)
}
