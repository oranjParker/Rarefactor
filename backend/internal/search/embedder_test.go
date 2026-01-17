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

func TestComputeEmbeddings_Error(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	e := &Embedder{baseURL: mockServer.URL, httpClient: &http.Client{}}

	_, err := e.ComputeEmbeddings(context.Background(), "test", false)
	if err == nil {
		t.Error("Expected error on 500 status, got nil")
	}

	// Test empty response
	mockServer2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data": []}`)
	}))
	defer mockServer2.Close()
	e.baseURL = mockServer2.URL
	_, err = e.ComputeEmbeddings(context.Background(), "test", false)
	if err == nil {
		t.Error("Expected error on empty data, got nil")
	}

	// Test malformed JSON
	mockServer3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `invalid json`)
	}))
	defer mockServer3.Close()
	e.baseURL = mockServer3.URL
	_, err = e.ComputeEmbeddings(context.Background(), "test", false)
	if err == nil {
		t.Error("Expected error on malformed JSON, got nil")
	}
}

func TestNewEmbedder_Default(t *testing.T) {
	os.Unsetenv("EMBEDDING_URL")
	e := NewEmbedder()
	if e.baseURL != "http://localhost:7997" {
		t.Errorf("Expected default URL, got %s", e.baseURL)
	}
}
