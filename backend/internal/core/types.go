package core

import (
	"context"
	"time"
)

type Document[T any] struct {
	ID             string         `json:"id"`
	ParentID       string         `json:"parent_id,omitempty"`
	Source         string         `json:"source"`
	Content        T              `json:"content"`
	CleanedContent T              `json:"cleaned_content,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	Depth          int            `json:"depth"`
	Ack            func()         `json:"-"`
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
