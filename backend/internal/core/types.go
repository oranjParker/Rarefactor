package core

import (
	"context"
	"time"
)

type Document[T any] struct {
	ID             string
	ParentID       string
	Source         string
	Content        T
	CleanedContent T
	Metadata       map[string]any
	CreatedAt      time.Time
	Depth          int
	Ack            func()
}

func (d *Document[T]) Clone() *Document[T] {
	if d == nil {
		return nil
	}

	newDoc := *d

	if d.Metadata != nil {
		newDoc.Metadata = make(map[string]any, len(d.Metadata))
		for k, v := range d.Metadata {
			newDoc.Metadata[k] = v
		}
	}

	return &newDoc
}

type Source[T any] interface {
	Stream(ctx context.Context) (<-chan T, error)
}

type Processor[In any, Out any] interface {
	Process(ctx context.Context, input In) ([]Out, error)
}

type FunctionalProcessor[In any, Out any] struct {
	Fn func(context.Context, In) ([]Out, error)
}

func (p *FunctionalProcessor[In, Out]) Process(ctx context.Context, input In) ([]Out, error) {
	return p.Fn(ctx, input)
}

type Sink[T any] interface {
	Write(ctx context.Context, item T) error
	Close() error
}

type Event struct {
	Type      string
	Timestamp time.Time
	Payload   any
}
