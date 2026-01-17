package crawler

import (
	"context"
	"testing"
	"time"

	pb "github.com/oranjParker/Rarefactor/generated/protos/v1"
)

type mockEngine struct {
	lastSeedURL   string
	lastNamespace string
	lastMaxDepth  int
	lastCrawlMode string
	called        chan struct{}
}

func (m *mockEngine) Run(ctx context.Context, seedURL, namespace string, maxDepth int, crawlMode string) {
	m.lastSeedURL = seedURL
	m.lastNamespace = namespace
	m.lastMaxDepth = maxDepth
	m.lastCrawlMode = crawlMode
	close(m.called)
}

func TestCrawl_Defaults(t *testing.T) {
	mock := &mockEngine{called: make(chan struct{})}

	s := &CrawlerServer{
		eng:       mock,
		serverCtx: context.Background(),
	}

	req := &pb.CrawlRequest{
		SeedUrl: "http://go.dev",
	}

	_, err := s.Crawl(context.Background(), req)
	if err != nil {
		t.Fatalf("Crawl failed: %v", err)
	}

	select {
	case <-mock.called:
	case <-time.After(1 * time.Second):
		t.Fatal("Engine.Run not called")
	}

	if mock.lastCrawlMode != "broad" {
		t.Errorf("Expected mode 'broad', got %q", mock.lastCrawlMode)
	}
	if mock.lastMaxDepth != 2 {
		t.Errorf("Expected maxDepth 2, got %d", mock.lastMaxDepth)
	}
}

func TestCrawl_Explicit(t *testing.T) {
	mock := &mockEngine{called: make(chan struct{})}
	s := &CrawlerServer{
		eng:       mock,
		serverCtx: context.Background(),
	}

	req := &pb.CrawlRequest{
		SeedUrl:   "http://example.com",
		MaxDepth:  5,
		CrawlMode: "targeted",
	}

	_, err := s.Crawl(context.Background(), req)
	if err != nil {
		t.Fatalf("Crawl failed: %v", err)
	}

	select {
	case <-mock.called:
	case <-time.After(1 * time.Second):
		t.Fatal("Engine.Run not called")
	}

	if mock.lastMaxDepth != 5 {
		t.Errorf("Expected maxDepth 5, got %d", mock.lastMaxDepth)
	}
	if mock.lastCrawlMode != "targeted" {
		t.Errorf("Expected mode 'targeted', got %q", mock.lastCrawlMode)
	}
}

func TestNewCrawlerServer(t *testing.T) {
	srv := NewCrawlerServer(context.Background(), nil, nil, nil, nil)
	if srv == nil {
		t.Fatal("Expected non-nil server")
	}
	if srv.eng == nil {
		t.Error("Expected engine to be initialized")
	}
	if srv.serverCtx == nil {
		t.Error("Expected serverCtx to be set")
	}
}
