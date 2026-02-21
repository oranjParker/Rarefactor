package processor

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/oranjParker/Rarefactor/internal/core"
)

type DiscoveryProcessor struct{}

func NewDiscoveryProcessor() *DiscoveryProcessor {
	return &DiscoveryProcessor{}
}

func (p *DiscoveryProcessor) Process(ctx context.Context, doc *core.Document[string]) ([]*core.Document[string], error) {
	log.Println("[Discovery] Processing document for links:", doc.ID)
	if (doc.Source != "web" && doc.Source != "discovery" && doc.Source != "api_trigger") || doc.Content == "" {
		return nil, nil
	}

	maxDepth := 0
	currentDepth := doc.Depth
	if val, ok := doc.Metadata["max_depth"]; ok {
		switch v := val.(type) {
		case int:
			maxDepth = v
		case float64:
			maxDepth = int(v)
		}
	}

	if currentDepth >= maxDepth && maxDepth > 0 {
		return nil, nil
	}
	reader := strings.NewReader(doc.Content)
	htmlDoc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML for discovery: %w", err)
	}

	var discoveredLinks []*core.Document[string]

	htmlDoc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		resolved := resolveURL(doc.ID, href)
		if resolved != "" && isLikelyHTML(resolved) {
			newDoc := &core.Document[string]{
				ID:        resolved,
				Source:    "discovery",
				Depth:     doc.Depth + 1,
				CreatedAt: time.Now(),
			}
			discoveredLinks = append(discoveredLinks, newDoc)
		}
	})

	log.Printf("[Discovery] Discovered %d links from %s.", len(discoveredLinks), doc.ID)

	return discoveredLinks, nil
}

func resolveURL(base, relative string) string {
	if len(relative) > 2048 {
		return ""
	}
	baseURL, err := url.Parse(base)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return ""
	}
	relURL, err := url.Parse(relative)
	if err != nil {
		return ""
	}
	resolved := baseURL.ResolveReference(relURL).String()
	if len(resolved) > 2048 {
		return ""
	}
	return resolved
}

func isLikelyHTML(u string) bool {
	lower := strings.ToLower(u)
	extensions := []string{".jpg", ".jpeg", ".png", ".gif", ".pdf", ".zip", ".webp", ".mp4", ".mp3", ".js", ".css"}
	for _, ext := range extensions {
		if strings.HasSuffix(lower, ext) {
			return false
		}
	}
	return strings.HasPrefix(lower, "http")
}
