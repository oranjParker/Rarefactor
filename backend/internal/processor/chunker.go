package processor

import (
	"context"
	"fmt"

	"github.com/oranjParker/Rarefactor/internal/core"
)

type ChunkerProcessor struct {
	MaxChunkSize int
	Overlap      int
}

func NewChunkerProcessor(size, overlap int) *ChunkerProcessor {
	return &ChunkerProcessor{MaxChunkSize: size, Overlap: overlap}
}

func (p *ChunkerProcessor) Process(ctx context.Context, doc *core.Document[string]) ([]*core.Document[string], error) {
	if doc.Metadata["is_chunk"] == true {
		return []*core.Document[string]{doc}, nil
	}

	text := doc.Content
	if len(text) <= p.MaxChunkSize {
		newDoc := doc.Clone()
		if newDoc.Metadata == nil {
			newDoc.Metadata = make(map[string]any)
		}
		newDoc.Metadata["is_chunk"] = true
		newDoc.Metadata["chunk_index"] = 0
		return []*core.Document[string]{newDoc}, nil
	}

	chunks := make([]*core.Document[string], 0)
	index := 0
	for i := 0; i < len(text); i += p.MaxChunkSize - p.Overlap {
		end := i + p.MaxChunkSize
		if end > len(text) {
			end = len(text)
		}

		chunkText := text[i:end]
		chunkDoc := doc.Clone()

		chunkDoc.ID = fmt.Sprintf("%s#chunk%d", doc.ID, index)
		chunkDoc.ParentID = doc.ID
		chunkDoc.Content = chunkText

		if chunkDoc.Metadata == nil {
			chunkDoc.Metadata = make(map[string]any)
		}
		chunkDoc.Metadata["is_chunk"] = true
		chunkDoc.Metadata["chunk_index"] = index
		chunkDoc.Metadata["chunk_size"] = len(chunkText)

		chunks = append(chunks, chunkDoc)
		index++

		if end == len(text) {
			break
		}
	}

	return chunks, nil
}
