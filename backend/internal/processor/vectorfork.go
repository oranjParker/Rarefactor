package processor

import (
	"context"
	"log"

	"github.com/oranjParker/Rarefactor/internal/core"
)

type VectorForkProcessor struct {
	VectorSink core.Sink[*core.Document[string]]
}

func NewVectorForkProcessor(sink core.Sink[*core.Document[string]]) *VectorForkProcessor {
	return &VectorForkProcessor{
		VectorSink: sink,
	}
}

func (p *VectorForkProcessor) Process(ctx context.Context, doc *core.Document[string]) ([]*core.Document[string], error) {
	if err := p.VectorSink.Write(ctx, doc); err != nil {
		log.Printf("[VectorFork] Warning: failed to push to vector queue: %v", err)
	}

	return []*core.Document[string]{doc}, nil
}
