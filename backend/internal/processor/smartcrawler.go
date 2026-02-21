package processor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oranjParker/Rarefactor/internal/core"
)

type SPAProcessor interface {
	Process(ctx context.Context, doc *core.Document[string]) ([]*core.Document[string], error)
}

type SmartCrawlerProcessor struct {
	Standard *CrawlerProcessor
	SPA      SPAProcessor
}

func NewSmartCrawlerProcessor() *SmartCrawlerProcessor {
	return &SmartCrawlerProcessor{
		Standard: NewCrawlerProcessor(),
		SPA:      NewSPACrawlerProcessor(60 * time.Second),
	}
}

func (p *SmartCrawlerProcessor) Process(ctx context.Context, doc *core.Document[string]) ([]*core.Document[string], error) {
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]any)
	}
	needsRender := false
	if val, ok := doc.Metadata["force_render"].(bool); ok && val {
		needsRender = true
	}

	if strings.Contains(doc.ID, "/app.") || strings.Contains(doc.ID, "dashboard") {
		needsRender = true
	}

	// SPA Heuristics: check for root markers in content if we already have it or just scan for specific extensions
	if strings.HasSuffix(doc.ID, ".js") || strings.HasSuffix(doc.ID, ".jsx") || strings.HasSuffix(doc.ID, ".tsx") {
		needsRender = true
	}

	if needsRender {
		fmt.Printf("[SmartCrawler] Using Headless Chrome for %s\n", doc.ID)
		doc.Metadata["crawler_type"] = "spa"
		return p.SPA.Process(ctx, doc)
	}
	doc.Metadata["crawler_type"] = "standard"

	results, err := p.Standard.Process(ctx, doc)

	if err == nil && len(results) > 0 {
		content := results[0].Content
		if len(content) < 200 || strings.Contains(content, "id=\"root\"") || strings.Contains(content, "id=\"app\"") {
			fmt.Printf("[SmartCrawler] SPA detected or content sparse, falling back to SPA render for %s\n", doc.ID)
			doc.Metadata["crawler_type"] = "spa"
			return p.SPA.Process(ctx, doc)
		}
	}

	return results, err
}
