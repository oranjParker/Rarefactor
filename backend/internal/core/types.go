package core

import (
	"context"
	"sync"
	"sync/atomic"
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
	CT             *CompletionTracker
}

type CompletionTracker struct {
	wg     sync.WaitGroup
	failed atomic.Bool
	ack    func()
	nack   func()
}

func NewCompletionTracker(ack, nack func()) *CompletionTracker {
	ct := &CompletionTracker{
		ack:  ack,
		nack: nack,
	}

	ct.failed.Store(false)
	return ct
}

func (ct *CompletionTracker) Add(delta int) {
	ct.wg.Add(delta)
}

func (ct *CompletionTracker) Done() {
	ct.wg.Done()
}

func (ct *CompletionTracker) Fail() {
	ct.failed.Store(true)
}

func (ct *CompletionTracker) WaitAndFinish() {
	ct.wg.Wait()
	if ct.failed.Load() {
		if ct.nack != nil {
			ct.nack()
		}
	} else {
		if ct.ack != nil {
			ct.ack()
		}
	}
}

func (d *Document[T]) Clone() *Document[T] {
	if d == nil {
		return nil
	}

	newDoc := *d

	if d.Metadata != nil {
		newDoc.Metadata = make(map[string]any, len(d.Metadata))
		for k, v := range d.Metadata {
			switch t := v.(type) {
			case []any:
				newDoc.Metadata[k] = append([]any(nil), t...)
			case []float32:
				newDoc.Metadata[k] = append([]float32(nil), t...)
			default:
				newDoc.Metadata[k] = v
			}
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

type Sink[T any] interface {
	Write(ctx context.Context, item T) error
	Close() error
}

type Event struct {
	Type      string
	Timestamp time.Time
	Payload   any
}
