package processor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/oranjParker/Rarefactor/internal/core"
)

type SPACrawlerProcessor struct {
	Timeout time.Duration
}

func NewSPACrawlerProcessor(timeout time.Duration) *SPACrawlerProcessor {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &SPACrawlerProcessor{Timeout: timeout}
}

func (p *SPACrawlerProcessor) Process(ctx context.Context, doc *core.Document[string]) ([]*core.Document[string], error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.Flag("blink-settings", "imagesEnabled=false"),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	taskCtx, cancel = context.WithTimeout(taskCtx, p.Timeout)
	defer cancel()

	newDoc := doc.Clone()
	if newDoc.Metadata == nil {
		newDoc.Metadata = make(map[string]any)
	}

	var html string
	err := chromedp.Run(taskCtx,
		chromedp.Navigate(newDoc.ID),
		chromedp.WaitVisible("body"),
		chromedp.OuterHTML("html", &html),
	)

	if err != nil {
		return nil, fmt.Errorf("SPA render failed for %s: %w", newDoc.ID, err)
	}

	htmlDoc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse rendered HTML: %w", err)
	}

	title := strings.TrimSpace(htmlDoc.Find("title").Text())
	newDoc.Metadata["title"] = title
	newDoc.Source = "web"
	newDoc.Metadata["is_spa_render"] = true
	newDoc.Metadata["crawled_at"] = time.Now().UTC().Unix()
	newDoc.Content = strings.Join(strings.Fields(htmlDoc.Find("h1, h2, h3, p, li, td, blockquote, article, main").Text()), " ")

	return []*core.Document[string]{newDoc}, nil
}
