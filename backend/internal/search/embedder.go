package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Embedder struct {
	baseURL    string
	httpClient *http.Client
}

func NewEmbedder() *Embedder {
	url := os.Getenv("EMBEDDING_URL")
	if url == "" {
		url = "http://localhost:7997/v1"
	}
	return &Embedder{
		baseURL:    url,
		httpClient: &http.Client{},
	}
}

type EmbedRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
	Task  string   `json:"task"`
}

type EmbedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (e *Embedder) ComputeEmbeddings(ctx context.Context, text string, isQuery bool) ([]float32, error) {
	task := "search_document"
	if isQuery {
		task = "search_query"
	}

	reqBody, _ := json.Marshal(EmbedRequest{
		Input: []string{text},
		Model: "nomic-ai/nomic-embed-text-v1.5",
		Task:  task,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/embeddings", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[Embedder] error status %d", resp.StatusCode)
	}

	var result EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}

	return result.Data[0].Embedding, nil
}
