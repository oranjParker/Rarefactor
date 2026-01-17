package search

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestComputeEmbeddings(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" || r.Method != "POST" {
			t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		response := `{
			"data": [
				{
					"embedding": [0.1, 0.2, 0.3, 0.4, 0.5]
				}
			]
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, response)
	}))
	defer mockServer.Close()

	os.Setenv("EMBEDDING_URL", mockServer.URL)
	defer os.Unsetenv("EMBEDDING_URL") // Cleanup

	embedder := NewEmbedder()

	vector, err := embedder.ComputeEmbeddings(context.Background(), "test query", true)
	if err != nil {
		t.Fatalf("ComputeEmbeddings failed: %v", err)
	}

	if len(vector) != 5 {
		t.Errorf("Expected vector length 5, got %d", len(vector))
	}
	if vector[0] != 0.1 {
		t.Errorf("Expected first element 0.1, got %f", vector[0])
	}
}
