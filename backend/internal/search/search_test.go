package search

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	pb "github.com/oranjParker/Rarefactor/generated/protos/v1"
	"github.com/qdrant/go-client/qdrant"
	"github.com/redis/go-redis/v9"
)

type MockEmbedder struct{}

func (m *MockEmbedder) ComputeEmbeddings(ctx context.Context, text string, isQuery bool) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}

type MockVectorDB struct {
	MockResults []*qdrant.ScoredPoint
}

func (m *MockVectorDB) Query(ctx context.Context, collection string, vector []float32, limit uint64) ([]*qdrant.ScoredPoint, error) {
	return m.MockResults, nil
}

func TestAutocomplete(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	ctx := context.Background()
	rdb.ZAdd(ctx, "rarefactor:autocomplete",
		redis.Z{Score: 0, Member: "apple"},
		redis.Z{Score: 0, Member: "application"},
		redis.Z{Score: 0, Member: "apply"},
		redis.Z{Score: 0, Member: "banana"},
	)

	rdb.ZAdd(ctx, "global_search_scores",
		redis.Z{Score: 100, Member: "apple"},
		redis.Z{Score: 10, Member: "apply"},
	)

	srv := NewSearchServer(nil, rdb, nil, nil)

	req := &pb.AutocompleteRequest{Prefix: "app", Limit: 5}
	resp, err := srv.Autocomplete(ctx, req)
	if err != nil {
		t.Fatalf("Autocomplete failed: %v", err)
	}

	if len(resp.Suggestions) != 3 {
		t.Errorf("Expected 3 suggestions, got %d", len(resp.Suggestions))
	}

	if resp.Suggestions[0] != "apple" {
		t.Errorf("Expected 'apple' first, got %s", resp.Suggestions[0])
	}
}

func TestSearch(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	mockQdrant := &MockVectorDB{
		MockResults: []*qdrant.ScoredPoint{
			{
				Id:    qdrant.NewID("1"),
				Score: 0.95,
				Payload: qdrant.NewValueMap(map[string]any{
					"url":   "https://go.dev",
					"title": "The Go Programming Language",
				}),
			},
		},
	}

	mockEmbedder := &MockEmbedder{}

	srv := NewSearchServer(nil, rdb, mockQdrant, mockEmbedder)

	ctx := context.Background()
	req := &pb.SearchRequest{Query: "golang", Limit: 10}

	resp, err := srv.Search(ctx, req)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(resp.Results))
	}
	if resp.Results[0].Title != "The Go Programming Language" {
		t.Errorf("Unexpected title: %s", resp.Results[0].Title)
	}

	score, err := rdb.ZScore(ctx, "global_search_scores", "golang").Result()
	if err != nil {
		t.Errorf("Failed to check score in Redis: %v", err)
	}
	if score != 1.0 {
		t.Errorf("Expected score 1.0 for new query, got %f", score)
	}
}

func TestAutocomplete_Empty(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	srv := NewSearchServer(nil, rdb, nil, nil)

	req := &pb.AutocompleteRequest{Prefix: "  "}
	resp, _ := srv.Autocomplete(context.Background(), req)
	if len(resp.Suggestions) != 0 {
		t.Error("Expected 0 suggestions for empty prefix")
	}

	req = &pb.AutocompleteRequest{Prefix: "none"}
	resp, _ = srv.Autocomplete(context.Background(), req)
	if len(resp.Suggestions) != 0 {
		t.Error("Expected 0 suggestions for unknown prefix")
	}
}

func TestUpdateDocument(t *testing.T) {
	srv := &SearchServer{}
	doc := &pb.Document{Url: "http://test.com", Title: "Test"}
	resp, err := srv.UpdateDocument(context.Background(), &pb.UpdateDocumentRequest{Document: doc})
	if err != nil {
		t.Fatalf("UpdateDocument failed: %v", err)
	}
	if resp.Url != doc.Url {
		t.Errorf("Expected URL %s, got %s", doc.Url, resp.Url)
	}
}

func TestSearch_Empty(t *testing.T) {
	srv := &SearchServer{}
	resp, err := srv.Search(context.Background(), &pb.SearchRequest{Query: ""})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(resp.Results) != 0 {
		t.Error("Expected 0 results for empty query")
	}
}
