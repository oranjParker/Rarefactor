package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/oranjParker/Rarefactor/internal/core"
)

type EmbeddingProcessor struct {
	Endpoint   string
	httpClient *http.Client
	Model      string
}

type EmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
	Task  string   `json:"task"`
}

type EmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func NewEmbeddingProcessor(endpoint string) *EmbeddingProcessor {
	return &EmbeddingProcessor{
		Endpoint: endpoint,
		Model:    "nomic-ai/nomic-embed-text-v1.5",
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (p *EmbeddingProcessor) Process(ctx context.Context, doc *core.Document[string]) ([]*core.Document[string], error) {
	newDoc := doc.Clone()
	if newDoc.Metadata == nil {
		newDoc.Metadata = make(map[string]any)
	}
	textToEmbed := newDoc.CleanedContent
	if textToEmbed == "" {
		textToEmbed = newDoc.Content
	}

	if textToEmbed == "" {
		return []*core.Document[string]{newDoc}, nil
	}

	reqBody, _ := json.Marshal(EmbeddingRequest{
		Input: []string{textToEmbed},
		Model: p.Model,
		Task:  "search_document",
	})

	url := p.Endpoint
	if !bytes.HasSuffix([]byte(url), []byte("/embeddings")) {
		url += "/embeddings"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding service unreachable at %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding service returned status %d", resp.StatusCode)
	}

	var embResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	if len(embResp.Data) > 0 {
		newDoc.Metadata["vector"] = embResp.Data[0].Embedding
	}

	return []*core.Document[string]{newDoc}, nil
}
