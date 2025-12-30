package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	generated "github.com/oranjParker/Rarefactor/generated/protos/v1"
	actors "github.com/oranjParker/Rarefactor/internal/crawler/actors"
)

const MaxBodySize = 2 * 1024 * 1024

type Engine struct {
	coordinator *actors.Coordinator
	concurrency int
}

func NewEngine(concurrency int, politeness time.Duration) *Engine {
	return &Engine{
		coordinator: actors.NewCoordinator(politeness),
		concurrency: concurrency,
	}
}

func (e *Engine) Run(seedURL string) {
	for i := 0; i < e.concurrency; i++ {
		go func() {
			for url := range e.coordinator.JobsChan {
				links, err := e.fetchAndParse(url)
				e.coordinator.ResultsChan <- actors.CrawlResult{URL: url, Links: links, Err: err}
			}
		}()
	}

	e.coordinator.AddURL(seedURL)

	var waitTimer *time.Timer

	for e.coordinator.HasWork() {
		var jobStream chan string
		var nextJob string

		url, ok := e.coordinator.GetNextJob()

		if ok {
			nextJob = url
			jobStream = e.coordinator.JobsChan
		}

		var waitChan <-chan time.Time
		if !ok && e.coordinator.ActiveWorkers() == 0 {
			waitTime := e.coordinator.TimeToNextJob()
			if waitTimer == nil {
				waitTimer = time.NewTimer(waitTime)
			} else {
				waitTimer.Reset(waitTime)
			}
			waitChan = waitTimer.C
		}

		select {
		case jobStream <- nextJob:
			e.coordinator.IncrementActiveWorkers()
		case res := <-e.coordinator.ResultsChan:
			e.coordinator.ProcessResult(res)
		case <-waitChan:

		}
	}

	close(e.coordinator.JobsChan)
}

func (e *Engine) fetchAndParse(url string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "RarefactorBot/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	limitReader := io.LimitReader(resp.Body, MaxBodySize)

	doc, err := goquery.NewDocumentFromReader(limitReader)
	if err != nil {
		return nil, err
	}

	doc.Find("script, style, nav, footer, header").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	title := doc.Find("title").Text()
	if title == "" {
		title = url
	}
	fmt.Printf("Crawled: %s\n", title)

	var links []string
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		absoluteUrl := resolveURL(url, href)
		if absoluteUrl != "" && strings.HasPrefix(url, "http") {
			links = append(links, absoluteUrl)
		}
	})

	return links, nil
}

func resolveURL(base, relative string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		return ""
	}
	relURL, err := url.Parse(relative)
	if err != nil {
		return ""
	}
	return baseURL.ResolveReference(relURL).String()
}

func (e *Engine) saveToPostgres(ctx context.Context, doc *generated.Document) (int, error) {
	var id int
	query := `
		INSERT INTO documents (url, domain, title, content, raw_html_size)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (url) DO UPDATE SET
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			crawled_at = CURRENT_TIMESTAMP
		RETURNING id;
	`

	u, _ := url.Parse(doc.Url)

	err := e.dbPool.QueryRow(ctx, query,
		doc.Url,
		u.Host,
		doc.Title,
		doc.Content,
		len(doc.Content),
	).Scan(&id)

	return id, err
}
