package core

import (
	"context"
	"fmt"
	"sync"
)

type Node[T any] struct {
	Name       string
	Processor  Processor[T, T]
	Downstream []*Node[T]
	Sink       Sink[T]
}

func (n *Node[T]) IsSink() bool {
	return n.Sink != nil
}

type GraphRunner[T any] struct {
	Name        string
	Source      Source[T]
	Nodes       map[string]*Node[T]
	Concurrency int
	wg          sync.WaitGroup
}

func NewGraphRunner[T any](name string, src Source[T], concurrency int) *GraphRunner[T] {
	if concurrency <= 0 {
		concurrency = 1
	}
	return &GraphRunner[T]{
		Name:        name,
		Source:      src,
		Nodes:       make(map[string]*Node[T]),
		Concurrency: concurrency,
	}
}

func (g *GraphRunner[T]) nodeExists(name string) bool {
	_, ok := g.Nodes[name]
	return ok
}

func (g *GraphRunner[T]) AddProcessor(name string, proc Processor[T, T]) error {
	if g.nodeExists(name) {
		return fmt.Errorf("node %s already exists in graph", name)
	}
	g.Nodes[name] = &Node[T]{Name: name, Processor: proc}
	return nil
}

func (g *GraphRunner[T]) AddSink(name string, sink Sink[T]) error {
	if g.nodeExists(name) {
		return fmt.Errorf("node %s already exists in graph", name)
	}
	g.Nodes[name] = &Node[T]{Name: name, Sink: sink}
	return nil
}

func (g *GraphRunner[T]) AddHybrid(name string, proc Processor[T, T], sink Sink[T]) error {
	if g.nodeExists(name) {
		return fmt.Errorf("node %s already exists in graph", name)
	}
	g.Nodes[name] = &Node[T]{Name: name, Processor: proc, Sink: sink}
	return nil
}

func (g *GraphRunner[T]) Connect(from, to string) error {
	f, ok1 := g.Nodes[from]
	t, ok2 := g.Nodes[to]
	if !ok1 || !ok2 {
		return fmt.Errorf("connection failed: node %s or %s not found", from, to)
	}
	f.Downstream = append(f.Downstream, t)
	return nil
}

func (g *GraphRunner[T]) Run(ctx context.Context) error {
	stream, err := g.Source.Stream(ctx)
	if err != nil {
		return fmt.Errorf("source error: %w", err)
	}

	startNode, ok := g.Nodes["start"]
	if !ok {
		return fmt.Errorf("graph execution error: no 'start' node found")
	}

	for i := 0; i < g.Concurrency; i++ {
		g.wg.Add(1)
		go func(workerID int) {
			defer g.wg.Done()
			for item := range stream {
				g.executeNode(ctx, startNode, item)
			}
		}(i)
	}

	g.wg.Wait()
	return nil
}

func (g *GraphRunner[T]) executeNode(ctx context.Context, node *Node[T], item T) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	currentItems := []T{item}

	if node.Processor != nil {
		results, err := node.Processor.Process(ctx, item)
		if err != nil {
			fmt.Printf("[%s] Processor Failure: %v\n]", node.Name, err)
		}
		currentItems = results
	}

	if node.Sink != nil {
		for _, resultItem := range currentItems {
			if err := node.Sink.Write(ctx, resultItem); err != nil {
				fmt.Printf("[%s] Sink error: %v\n", node.Name, err)
			}
		}
	}

	for _, res := range currentItems {
		if len(node.Downstream) > 1 {
			var executionGroup sync.WaitGroup
			for i := 0; i < len(node.Downstream); i++ {
				executionGroup.Add(1)
				var passItem T
				if cloner, ok := any(res).(interface{ Clone() T }); ok {
					passItem = cloner.Clone()
				} else {
					passItem = res
				}
				go func(i int, passItem T) {
					defer executionGroup.Done()
					g.executeNode(ctx, node.Downstream[i], passItem)
				}(i, passItem)
			}
			executionGroup.Wait()
		} else if len(node.Downstream) == 1 {
			g.executeNode(ctx, node.Downstream[0], res)
		}
	}
}
