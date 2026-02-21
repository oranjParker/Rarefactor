package processor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/oranjParker/Rarefactor/internal/core"
	"github.com/oranjParker/Rarefactor/internal/utils"
)

type CrawlerProcessor struct {
	client *http.Client
}

func NewCrawlerProcessor() *CrawlerProcessor {
	return &CrawlerProcessor{
		client: utils.NewSafeHTTPClient(utils.ClientConfig{Timeout: 10 * time.Second}),
	}
}

func (p *CrawlerProcessor) Process(ctx context.Context, doc *core.Document[string]) ([]*core.Document[string], error) {
	urlStr := doc.ID
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "RarefactorBot/2.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, err
	}

	htmlDoc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	htmlDoc.Find("script, style, noscript, iframe, svg, nav, footer").Remove()

	cleanedHTML, err := htmlDoc.Html()
	if err != nil {
		return nil, fmt.Errorf("html re-render failed: %w", err)
	}

	title := strings.TrimSpace(htmlDoc.Find("title").Text())

	newDoc := doc.Clone()
	if newDoc.Metadata == nil {
		newDoc.Metadata = make(map[string]any)
	}
	newDoc.Content = cleanedHTML
	newDoc.Source = "web"
	newDoc.Metadata["title"] = title
	newDoc.Metadata["http_status"] = resp.StatusCode
	newDoc.Metadata["crawled_at"] = time.Now().UTC().Unix()

	return []*core.Document[string]{newDoc}, nil
}
