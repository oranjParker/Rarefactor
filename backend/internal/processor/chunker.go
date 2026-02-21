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

	if len(delimiters) == 0 {
		var chunks []string
		runes := []rune(text)
		step := p.MaxChunkSize - p.Overlap
		if step <= 0 {
			step = p.MaxChunkSize
		}
		for i := 0; i < len(runes); i += step {
			end := i + p.MaxChunkSize
			if end > len(runes) {
				end = len(runes)
			}
			chunks = append(chunks, string(runes[i:end]))
			if end == len(runes) {
				break
			}
		}
		return chunks
	}

	delimiter := delimiters[0]
	parts := strings.Split(text, delimiter)
	var result []string
	var current strings.Builder

	for i, part := range parts {
		partWithDelimiter := part
		if i < len(parts)-1 {
			partWithDelimiter += delimiter
		}

		if len(partWithDelimiter) > p.MaxChunkSize {
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
			subChunks := p.splitRecursive(partWithDelimiter, delimiters[1:])
			result = append(result, subChunks...)
		} else if current.Len()+len(partWithDelimiter) <= p.MaxChunkSize {
			current.WriteString(partWithDelimiter)
		} else {
			if current.Len() > 0 {
				result = append(result, current.String())
			}
			current.Reset()
			current.WriteString(partWithDelimiter)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}
