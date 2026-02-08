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
}

type EmbeddingRequest struct {
	Text string `json:"text"`
}

type EmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

func NewEmbeddingProcessor(endpoint string) *EmbeddingProcessor {
	return &EmbeddingProcessor{
		Endpoint: endpoint,
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

	reqBody, _ := json.Marshal(EmbeddingRequest{Text: textToEmbed})
	req, err := http.NewRequestWithContext(ctx, "POST", p.Endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding service returned status %d", resp.StatusCode)
	}

	var embResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode embedding: %w", err)
	}

	newDoc.Metadata["vector"] = embResp.Embedding

	return []*core.Document[string]{newDoc}, nil
}
