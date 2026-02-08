package processor

import (
	"context"
	"strings"

	"github.com/oranjParker/Rarefactor/internal/core"
)

type EnrichmentProcessor struct {
	StopWords map[string]struct{}
}

func NewEnrichmentProcessor() *EnrichmentProcessor {
	swList := []string{
		"i", "me", "my", "myself", "we", "our", "ours", "ourselves", "you", "your", "yours",
		"yourself", "yourselves", "he", "him", "his", "himself", "she", "her", "hers",
		"herself", "it", "its", "itself", "they", "them", "their", "theirs", "themselves",
		"what", "which", "who", "whom", "this", "that", "these", "those", "am", "is", "are",
		"was", "were", "be", "been", "being", "have", "has", "had", "having", "do", "does",
		"did", "doing", "a", "an", "the", "and", "but", "if", "or", "because", "as", "until",
		"while", "of", "at", "by", "for", "with", "about", "against", "between", "into",
		"through", "during", "before", "after", "above", "below", "to", "from", "up", "down",
		"in", "out", "on", "off", "over", "under", "again", "further", "then", "once", "here",
		"there", "when", "where", "why", "how", "all", "any", "both", "each", "few", "more",
		"most", "other", "some", "such", "no", "nor", "only", "own", "same", "so", "than",
		"too", "very", "s", "t", "can", "will", "just", "don", "should", "now",
	}

	stopMap := make(map[string]struct{})
	for _, word := range swList {
		stopMap[word] = struct{}{}
	}
	return &EnrichmentProcessor{StopWords: stopMap}
}

func (p *EnrichmentProcessor) Process(ctx context.Context, doc *core.Document[string]) ([]*core.Document[string], error) {
	newDoc := doc.Clone()
	cleaned := strings.ToLower(newDoc.Content)
	cleaned = strings.ReplaceAll(cleaned, "can't", "cannot")
	cleaned = strings.ReplaceAll(cleaned, "n't", " not")
	cleaned = strings.ReplaceAll(cleaned, "it's", "it is")
	cleaned = strings.ReplaceAll(cleaned, "i'm", "i am")
	cleaned = strings.ReplaceAll(cleaned, "you're", "you are")

	newDoc.CleanedContent = cleaned
	if newDoc.Metadata == nil {
		newDoc.Metadata = make(map[string]any)
	}
	newDoc.Metadata["enriched"] = true

	return []*core.Document[string]{newDoc}, nil
}
