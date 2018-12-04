package test

import (
	"context"
	"time"
)

const randomContextKey = "controlled_random"

type NamedLogger interface {
	Log(args ...interface{})
	Name() string
}

func WithContext(f func(ctx context.Context)) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f(ctx)
}

func WithContextWithRand(t NamedLogger, f func(ctx context.Context, ctrlRand *ControlledRand)) {
	ctrlRand := NewControlledRand(t)
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), randomContextKey, ctrlRand))
	defer cancel()
	f(ctx, ctrlRand)
}

func WithContextWithTimeout(t NamedLogger, d time.Duration, f func(ctx context.Context)) {
	ctrlRand := NewControlledRand(t)
	ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), randomContextKey, ctrlRand), d)
	defer cancel()
	f(ctx)
}

func GetRand(ctx context.Context) *ControlledRand {
	result := ctx.Value(randomContextKey).(*ControlledRand)
	if result == nil {
		panic("cant find controlled number object in context")
	}
	return result
}
