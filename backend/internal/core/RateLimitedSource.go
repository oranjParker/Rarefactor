package core

import (
	"context"
	"time"
)

type RateLimitedSource[T any] struct {
	Src      Source[T]
	Interval time.Duration
}

func NewRateLimitedSource[T any](src Source[T], interval time.Duration) *RateLimitedSource[T] {
	return &RateLimitedSource[T]{Src: src, Interval: interval}
}

func (s *RateLimitedSource[T]) Stream(ctx context.Context) (<-chan T, error) {
	srcStream, err := s.Src.Stream(ctx)
	if err != nil {
		return nil, err
	}

	out := make(chan T)
	ticker := time.NewTicker(s.Interval)

	go func() {
		defer ticker.Stop()
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			case item, ok := <-srcStream:
				if !ok {
					return
				}
				select {
				case <-ticker.C:
					select {
					case out <- item:
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}
