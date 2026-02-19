package processor

import (
	"context"
	"fmt"
	"strings"

	"github.com/oranjParker/Rarefactor/internal/core"
)

type ChunkerProcessor struct {
	MaxChunkSize int
	Overlap      int
	Delimiters   []string
}

func NewChunkerProcessor(size, overlap int) *ChunkerProcessor {
	return &ChunkerProcessor{
		MaxChunkSize: size,
		Overlap:      overlap,
		Delimiters:   []string{"\n\n", "\n", ". ", "! ", "? ", ";", ":", " "},
	}
}

func (p *ChunkerProcessor) Process(ctx context.Context, doc *core.Document[string]) ([]*core.Document[string], error) {
	if doc.Content == "" {
		return nil, nil
	}
	if val, ok := doc.Metadata["is_chunk"].(bool); ok && val {
		return []*core.Document[string]{doc.Clone()}, nil
	}

	rawChunks := p.splitRecursive(doc.Content, p.Delimiters)

	var processedChunks []*core.Document[string]
	for i, chunkText := range rawChunks {
		if strings.TrimSpace(chunkText) == "" {
			continue
		}

		chunkID := fmt.Sprintf("%s#chunk%d", doc.ID, i)
		newDoc := doc.Clone()
		newDoc.ID = chunkID
		newDoc.ParentID = doc.ID
		newDoc.Content = chunkText

		newDoc.CleanedContent = ""

		if newDoc.Metadata == nil {
			newDoc.Metadata = make(map[string]any)
		}
		newDoc.Metadata["is_chunk"] = true
		newDoc.Metadata["chunk_index"] = i
		newDoc.Metadata["chunk_size"] = len(chunkText)

		processedChunks = append(processedChunks, newDoc)
	}

	return processedChunks, nil
}

func (p *ChunkerProcessor) splitRecursive(text string, delimiters []string) []string {
	if len(text) <= p.MaxChunkSize {
		return []string{text}
	}

	var chunks []string
	if len(delimiters) == 0 {
		runes := []rune(text)
		for i := 0; i < len(runes); i += p.MaxChunkSize - p.Overlap {
			end := i + p.MaxChunkSize
			if end > len(runes) {
				end = len(runes)
			}
			chunks = append(chunks, string(runes[i:end]))
		}
		return chunks
	}

	delimiter := delimiters[0]
	parts := strings.Split(text, delimiter)

	var finalChunks []string
	var currentChunk strings.Builder

	for i, part := range parts {
		if len(part) > p.MaxChunkSize {
			if currentChunk.Len() > 0 {
				finalChunks = append(finalChunks, strings.TrimSpace(currentChunk.String()))
				currentChunk.Reset()
			}

			subChunks := p.splitRecursive(part, delimiters[1:])
			finalChunks = append(finalChunks, subChunks...)
			continue
		}

		partValue := part
		if i < len(parts)-1 {
			partValue += delimiter
		}

		if currentChunk.Len()+len(partValue) <= p.MaxChunkSize {
			if currentChunk.Len() > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
			}

			if len(partValue) > p.MaxChunkSize {
				subChunks := p.splitRecursive(partValue, delimiters[1:])
				chunks = append(chunks, subChunks...)
			} else {
				currentChunk.WriteString(partValue)
			}
		} else {
			currentChunk.WriteString(partValue)
		}
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}
	return chunks
}
