package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// =========================================================================
// MOCKS FOR GRAPH TESTING
// =========================================================================

type mockSource struct {
	items []string
}

func (m *mockSource) Stream(ctx context.Context) (<-chan string, error) {
	ch := make(chan string)
	go func() {
		defer close(ch)
		for _, item := range m.items {
			select {
			case ch <- item:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

type mockProcessor struct {
	suffix string
	err    error
}

func (p *mockProcessor) Process(ctx context.Context, in string) ([]string, error) {
	if p.err != nil {
		return nil, p.err
	}
	return []string{in + p.suffix}, nil
}

type mockSink struct {
	received []string
	mu       sync.Mutex
}

func (s *mockSink) Write(ctx context.Context, item string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.received = append(s.received, item)
	return nil
}

func (s *mockSink) Close() error { return nil }

// =========================================================================
// TESTS
// =========================================================================

func TestDocument_Clone(t *testing.T) {
	t.Run("Clone Nil", func(t *testing.T) {
		var d *Document[string]
		if d.Clone() != nil {
			t.Error("cloning nil document should return nil")
		}
	})

	t.Run("Clone with Metadata", func(t *testing.T) {
		original := &Document[string]{
			ID:       "id-1",
			Metadata: map[string]any{"deep": "value"},
		}

		clone := original.Clone()
		clone.Metadata["deep"] = "changed"

		if original.Metadata["deep"] == "changed" {
			t.Errorf("cloning failed: metadata map points to same memory location")
		}
	})

	t.Run("Clone without Metadata", func(t *testing.T) {
		original := &Document[string]{ID: "id-1"}
		clone := original.Clone()
		if clone.ID != "id-1" {
			t.Error("cloning failed to copy ID")
		}
		if clone.Metadata != nil {
			t.Error("cloned metadata should be nil if original was nil")
		}
	})
}

func TestDocument_DoAck(t *testing.T) {
	acked := false
	doc := &Document[string]{
		Ack: func() { acked = true },
	}
	doc.DoAck()
	if !acked {
		t.Error("DoAck failed to call Ack function")
	}

	// Should not panic
	var nilDoc *Document[string]
	nilDoc.DoAck()

	docNoAck := &Document[string]{}
	docNoAck.DoAck()
}

func TestGraphRunner_TopologyErrors(t *testing.T) {
	src := &mockSource{items: []string{"test"}}
	runner := NewGraphRunner("test-graph", src, 1)

	t.Run("Duplicate Node ID", func(t *testing.T) {
		_ = runner.AddProcessor("node1", &mockProcessor{})
		if err := runner.AddProcessor("node1", &mockProcessor{}); err == nil {
			t.Error("expected error when adding duplicate node ID")
		}
	})

	t.Run("Missing Source Connection", func(t *testing.T) {
		if err := runner.Connect("missing", "target"); err == nil {
			t.Error("expected error when connecting from non-existent node")
		}
	})

	t.Run("Run Without Start Node", func(t *testing.T) {
		err := runner.Run(context.Background())
		if err == nil || !strings.Contains(err.Error(), "no 'start' node") {
			t.Errorf("expected error about missing start node, got: %v", err)
		}
	})
}

func TestGraphRunner_HybridNode(t *testing.T) {
	ctx := context.Background()
	src := &mockSource{items: []string{"input"}}
	runner := NewGraphRunner[string]("hybrid-test", src, 1)

	intermediateSink := &mockSink{}
	finalSink := &mockSink{}

	_ = runner.AddHybrid("start", &mockProcessor{suffix: "-proc"}, intermediateSink)
	_ = runner.AddSink("end", finalSink)
	_ = runner.Connect("start", "end")

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(intermediateSink.received) != 1 || intermediateSink.received[0] != "input-proc" {
		t.Errorf("Intermediate sink failed. Got: %v", intermediateSink.received)
	}

	if len(finalSink.received) != 1 || finalSink.received[0] != "input-proc" {
		t.Errorf("Final sink failed to receive downstream item. Got: %v", finalSink.received)
	}
}

func TestGraphRunner_ExecutionFlow(t *testing.T) {
	ctx := context.Background()
	src := &mockSource{items: []string{"a", "b"}}
	runner := NewGraphRunner[string]("flow-test", src, 1)

	sink := &mockSink{}

	_ = runner.AddProcessor("start", &mockProcessor{suffix: "-1"})
	_ = runner.AddProcessor("p2", &mockProcessor{suffix: "-2"})
	_ = runner.AddSink("end", sink)

	_ = runner.Connect("start", "p2")
	_ = runner.Connect("p2", "end")

	err := runner.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(sink.received) != 2 {
		t.Fatalf("expected 2 items in sink, got %d", len(sink.received))
	}

	expected := []string{"a-1-2", "b-1-2"}
	for i, v := range sink.received {
		if v != expected[i] {
			t.Errorf("mismatch at index %d: expected %s, got %s", i, expected[i], v)
		}
	}
}

func TestGraphRunner_ConcurrencySafety(t *testing.T) {
	itemCount := 100
	items := make([]string, itemCount)
	for i := 0; i < itemCount; i++ {
		items[i] = fmt.Sprintf("item-%d", i)
	}

	src := &mockSource{items: items}
	runner := NewGraphRunner[string]("stress-test", src, 10)
	sink := &mockSink{}

	_ = runner.AddProcessor("start", &mockProcessor{suffix: ""})
	_ = runner.AddSink("end", sink)
	_ = runner.Connect("start", "end")

	if err := runner.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(sink.received) != itemCount {
		t.Errorf("lost items during concurrent run: expected %d, got %d", itemCount, len(sink.received))
	}
}

func TestGraphRunner_ErrorPaths(t *testing.T) {
	ctx := context.Background()

	t.Run("Source Stream Error", func(t *testing.T) {
		src := &mockSourceErr{err: fmt.Errorf("stream fail")}
		runner := NewGraphRunner("err-test", src, 1)
		err := runner.Run(ctx)
		if err == nil || !strings.Contains(err.Error(), "source error") {
			t.Errorf("expected source error, got: %v", err)
		}
	})

	t.Run("Processor Error", func(t *testing.T) {
		src := &mockSource{items: []string{"input"}}
		runner := NewGraphRunner("proc-err", src, 1)
		_ = runner.AddProcessor("start", &mockProcessor{err: fmt.Errorf("proc fail")})
		_ = runner.Run(ctx) // Should not return error, just log it
	})

	t.Run("Sink Error", func(t *testing.T) {
		src := &mockSource{items: []string{"input"}}
		runner := NewGraphRunner("sink-err", src, 1)
		_ = runner.AddSink("start", &mockSinkErr{err: fmt.Errorf("sink fail")})
		_ = runner.Run(ctx) // Should not return error, just log it
	})
}

func TestGraphRunner_CloneDownstream(t *testing.T) {
	src := &mockSource{items: []string{"input"}}
	runner := NewGraphRunner("clone-test", src, 1)

	sink1 := &mockSink{}
	sink2 := &mockSink{}

	_ = runner.AddProcessor("start", &mockProcessor{suffix: ""})
	_ = runner.AddSink("s1", sink1)
	_ = runner.AddSink("s2", sink2)

	_ = runner.Connect("start", "s1")
	_ = runner.Connect("start", "s2")

	// Use Document which has Clone() method
	srcDoc := &mockSourceDoc{items: []*Document[string]{{ID: "doc1"}}}
	runnerDoc := NewGraphRunner[*Document[string]]("doc-test", srcDoc, 1)
	_ = runnerDoc.AddProcessor("start", &mockProcessorDoc{})
	_ = runnerDoc.AddSink("s1", &mockSinkDoc{sink: sink1})
	_ = runnerDoc.AddSink("s2", &mockSinkDoc{sink: sink2})
	_ = runnerDoc.Connect("start", "s1")
	_ = runnerDoc.Connect("start", "s2")

	_ = runnerDoc.Run(context.Background())

	if len(sink1.received) != 1 || len(sink2.received) != 1 {
		t.Errorf("cloning downstream failed to deliver items: %d, %d", len(sink1.received), len(sink2.received))
	}
}

type mockSourceErr struct {
	err error
}

func (m *mockSourceErr) Stream(ctx context.Context) (<-chan string, error) {
	return nil, m.err
}

type mockSinkErr struct {
	err error
}

func (s *mockSinkErr) Write(ctx context.Context, item string) error { return s.err }
func (s *mockSinkErr) Close() error                                 { return nil }

type mockSourceDoc struct {
	items []*Document[string]
}

func (m *mockSourceDoc) Stream(ctx context.Context) (<-chan *Document[string], error) {
	ch := make(chan *Document[string])
	go func() {
		defer close(ch)
		for _, item := range m.items {
			ch <- item
		}
	}()
	return ch, nil
}

type mockProcessorDoc struct{}

func (p *mockProcessorDoc) Process(ctx context.Context, in *Document[string]) ([]*Document[string], error) {
	return []*Document[string]{in}, nil
}

type mockSinkDoc struct {
	sink *mockSink
}

func (s *mockSinkDoc) Write(ctx context.Context, item *Document[string]) error {
	return s.sink.Write(ctx, item.ID)
}
func (s *mockSinkDoc) Close() error { return nil }
