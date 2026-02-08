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
	)
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	taskCtx, cancel = context.WithTimeout(taskCtx, p.Timeout)
	defer cancel()

	var html string
	err := chromedp.Run(taskCtx,
		chromedp.Navigate(doc.ID),
		chromedp.Sleep(3*time.Second),
		chromedp.OuterHTML("html", &html),
	)

	if err != nil {
		return nil, fmt.Errorf("SPA render failed for %s: %w", doc.ID, err)
	}

	htmlDoc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse rendered HTML: %w", err)
	}

	doc.Metadata["title"] = strings.TrimSpace(htmlDoc.Find("title").Text())

	htmlDoc.Find("script, style, nav, footer, header, meta, noscript, iframe, svg").Remove()

	doc.Content = strings.Join(strings.Fields(htmlDoc.Find("body").Text()), " ")

	return []*core.Document[string]{doc}, nil
}
