package core

import (
	"context"
	"fmt"
	"sync"
)

type PipelineRunner[T any] struct {
	Source     Source[T]
	Processors []Processor[T, T]
	Sink       Sink[T]
	Config     PipelineConfig
}

type PipelineConfig struct {
	Concurrency int
	Name        string
}

func NewPipelineRunner[T any](src Source[T], sink Sink[T], cfg PipelineConfig) *PipelineRunner[T] {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}
	return &PipelineRunner[T]{
		Source:     src,
		Sink:       sink,
		Config:     cfg,
		Processors: make([]Processor[T, T], 0),
	}
}

func (p *PipelineRunner[T]) AddProcessor(proc Processor[T, T]) {
	p.Processors = append(p.Processors, proc)
}

func (p *PipelineRunner[T]) Run(ctx context.Context) error {
	stream, err := p.Source.Stream(ctx)
	if err != nil {
		return fmt.Errorf("pipeline [%s] source error: %w", p.Config.Name, err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, p.Config.Concurrency)

	for i := 0; i < p.Config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for item := range stream {
				select {
				case <-ctx.Done():
					return
				default:
				}

				processedItems, err := p.processRecursive(ctx, item, 0)
				if err != nil {
					fmt.Printf("Worker %d error: %v\n", workerID, err)
					continue
				}

				for _, processedItem := range processedItems {
					if err := p.Sink.Write(ctx, processedItem); err != nil {
						select {
						case errChan <- fmt.Errorf("sink write error: %w", err):
						default:
						}
						return
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)
	if len(errChan) > 0 {
		return <-errChan
	}

	return nil
}

func (p *PipelineRunner[T]) processRecursive(ctx context.Context, item T, procIdx int) ([]T, error) {
	if procIdx >= len(p.Processors) {
		return []T{item}, nil
	}
	expanded, err := p.Processors[procIdx].Process(ctx, item)
	if err != nil {
		return nil, err
	}

	finalResults := make([]T, 0)
	for _, nextItem := range expanded {
		nextResults, err := p.processRecursive(ctx, nextItem, procIdx+1)
		if err != nil {
			return nil, err
		}
		finalResults = append(finalResults, nextResults...)
	}

	return finalResults, nil
}
