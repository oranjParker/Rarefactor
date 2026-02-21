package sink

import (
	"context"
	"fmt"

	"github.com/oranjParker/Rarefactor/internal/core"
	"github.com/oranjParker/Rarefactor/internal/database"
)

type QdrantSink struct {
	client     *database.QdrantClient
	collection string
}

func NewQdrantSink(client *database.QdrantClient, collection string) *QdrantSink {
	return &QdrantSink{
		client:     client,
		collection: collection,
	}
}

func (s *QdrantSink) Write(ctx context.Context, doc *core.Document[string]) error {
	val, ok := doc.Metadata["vector"]
	if !ok {
		return fmt.Errorf("document %s missing vector data", doc.ID)
	}

	vector, ok := val.([]float32)
	if !ok {
		return fmt.Errorf("invalid vector type for document %s", doc.ID)
	}

	summary := doc.Content
	if s, ok := doc.Metadata["summary"].(string); ok {
		summary = s
	}

	title := ""
	if t, ok := doc.Metadata["title"].(string); ok {
		title = t
	}

	err := s.client.Upsert(
		ctx,
		s.collection,
		doc.ID,
		title,
		summary,
		vector,
	)
	if err != nil {
		return fmt.Errorf("qdrant upsert failed: %w", err)
	}

	return nil
}

func (s *QdrantSink) Close() error {
	return nil
}
