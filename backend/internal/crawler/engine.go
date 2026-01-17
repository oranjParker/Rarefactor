package crawler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jimsmart/grobotstxt"
	"github.com/oranjParker/Rarefactor/internal/database"
	"github.com/oranjParker/Rarefactor/internal/search"
	"github.com/oranjParker/Rarefactor/internal/utils"
	"github.com/redis/go-redis/v9"
)

const (
	MaxBodySize    = 2 * 1024 * 1024
	UserAgent      = "RarefactorBot/1.0 (+https://github.com/oranjParker/Rarefactor)"
	RobotsCacheTTL = 24 * time.Hour
)

type DocumentStorage interface {
	PersistDocument(ctx context.Context, urlStr, title, content, namespace string) error
}

type EngineRunner interface {
	Run(ctx context.Context, seedURL, namespace string, maxDepth int, crawlMode string)
}

type Engine struct {
	coordinator  *Coordinator
	concurrency  int
	dbPool       *pgxpool.Pool
	redisClient  *redis.Client
	qdrantClient *database.QdrantClient
	embedder     *search.Embedder
	httpClient   *http.Client
	storage      DocumentStorage
}

func NewEngine(db *pgxpool.Pool, concurrency int, politeness time.Duration, redisClient *redis.Client,
	qdb *database.QdrantClient, emb *search.Embedder) *Engine {
	e := &Engine{
		dbPool:       db,
		coordinator:  NewCoordinator(politeness),
		concurrency:  concurrency,
		redisClient:  redisClient,
		qdrantClient: qdb,
		embedder:     emb,
		httpClient: utils.NewSafeHTTPClient(utils.ClientConfig{
			Timeout:       10 * time.Second,
			AllowInternal: false,
		}),
	}
	e.storage = e // Engine implements DocumentStorage
	return e
}

func (e *Engine) Run(ctx context.Context, seedURL, namespace string, maxDepth int, crawlMode string) {
	log.Printf("[Engine] Starting crawl: %s (mode: %s, maxDepth: %d)", seedURL, crawlMode, maxDepth)

	// Determine effective maxDepth and domain restriction based on mode
	effectiveMaxDepth := maxDepth
	var allowedDomain string

	if crawlMode == "broad" {
		effectiveMaxDepth = 2
	} else if crawlMode == "targeted" {
		effectiveMaxDepth = 10
		var err error
		allowedDomain, err = utils.GetBaseDomain(seedURL)
		if err != nil {
			log.Printf("[Engine] Failed to get base domain for targeted crawl: %v", err)
			return
		}
	}

	for i := 0; i < e.concurrency; i++ {
		go func(id int) {
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-e.coordinator.JobsChan:
					if !ok {
						return
					}

					allowed, _ := e.isAllowed(ctx, job.URL)
					if !allowed {
						e.coordinator.ResultsChan <- CrawlResult{
							URL:           job.URL,
							Depth:         job.Depth,
							MaxDepth:      job.MaxDepth,
							AllowedDomain: job.AllowedDomain,
							Err:           fmt.Errorf("robots not permitted"),
						}
						continue
					}

					links, title, content, err := e.fetchAndParse(ctx, job.URL)
					if err == nil {
						_ = e.storage.PersistDocument(ctx, job.URL, title, content, namespace)
					}

					e.coordinator.ResultsChan <- CrawlResult{
						URL:           job.URL,
						Depth:         job.Depth,
						MaxDepth:      job.MaxDepth,
						AllowedDomain: job.AllowedDomain,
						Links:         links,
						Title:         title,
						Content:       content,
						Err:           err,
					}
				}
			}
		}(i)
	}

	e.coordinator.AddURL(seedURL, 0, effectiveMaxDepth, allowedDomain)

	var waitTimer *time.Timer

	for e.coordinator.HasWork() {
		var jobStream chan URLJob
		var nextJob URLJob

		job, ok := e.coordinator.GetNextJob()

		if ok {
			nextJob = job
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
	log.Printf("[Engine] Crawl finished for seed: %s", seedURL)
}

func (e *Engine) fetchAndParse(ctx context.Context, url string) ([]string, string, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, "", "", err
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, "", "", err
	}

	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, "", "", fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return nil, "", "", fmt.Errorf("non-html content")
	}

	limitReader := io.LimitReader(resp.Body, MaxBodySize)

	doc, err := goquery.NewDocumentFromReader(limitReader)
	if err != nil {
		return nil, "", "", err
	}

	doc.Find("script, style, nav, footer, header").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	title := strings.TrimSpace(doc.Find("title").Text())
	if title == "" {
		title = url
	}
	content := strings.TrimSpace(doc.Find("body").Text())
	content = strings.Join(strings.Fields(content), " ")

	var links []string
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		absoluteUrl := resolveURL(url, href)
		if absoluteUrl != "" && isLikelyHTML(absoluteUrl) {
			links = append(links, absoluteUrl)
		}
	})

	fmt.Printf("Crawled: %s\n", title)
	return links, title, content, nil
}

func (e *Engine) isAllowed(ctx context.Context, targetURL string) (bool, error) {
	u, err := url.Parse(targetURL)
	if err != nil {
		return false, err
	}

	cacheKey := fmt.Sprintf("robots:%s", u.Host)
	var robotsData string
	var rerr error

	if e.redisClient != nil {
		robotsData, rerr = e.redisClient.Get(ctx, cacheKey).Result()
	} else {
		rerr = redis.Nil
	}

	if rerr == redis.Nil {
		robotsURL := fmt.Sprintf("%s://%s/robots.txt", u.Scheme, u.Host)
		req, err := http.NewRequestWithContext(ctx, "GET", robotsURL, nil)
		if err != nil {
			return true, nil
		}
		req.Header.Set("User-Agent", UserAgent)

		resp, err := e.httpClient.Do(req)
		if err != nil {
			return false, nil
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			return false, nil // Fail-closed on server error
		}

		if resp.StatusCode == 404 {
			if e.redisClient != nil {
				e.redisClient.Set(ctx, cacheKey, "", time.Hour)
			}
			return true, nil
		}

		body, _ := io.ReadAll(resp.Body)
		robotsData = string(body)
		if e.redisClient != nil {
			e.redisClient.Set(ctx, cacheKey, robotsData, RobotsCacheTTL)
		}
	} else if rerr != nil {
		return true, nil // Fallback
	}

	return grobotstxt.AgentAllowed(robotsData, UserAgent, u.Path), nil
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

func resolveURL(base, relative string) string {
	if len(relative) > 2048 {
		return ""
	}
	baseURL, err := url.Parse(base)
	if err != nil {
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

func (e *Engine) PersistDocument(ctx context.Context, urlStr, title, content, namespace string) error {
	return e.persistDocument(ctx, urlStr, title, content, namespace)
}

func (e *Engine) persistDocument(ctx context.Context, urlStr, title, content, namespace string) error {
	if e.dbPool == nil {
		return nil
	}
	query := `
		INSERT INTO documents (url, domain, title, content, raw_content_size, namespace)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (url) DO UPDATE SET
			title = EXCLUDED.title,
			content = EXCLUDED.content,
		    raw_content_size = EXCLUDED.raw_content_size,
			crawled_at = CURRENT_TIMESTAMP
	`

	u, _ := url.Parse(urlStr)

	// Truncate and sanitize title and content
	title = utils.SanitizeUTF8(title)
	if len(title) > 500 {
		title = title[:500]
	}
	content = utils.SanitizeUTF8(content)
	if len(content) > 100000 {
		content = content[:100000]
	}

	_, err := e.dbPool.Exec(ctx, query,
		urlStr,
		u.Host,
		title,
		content,
		len(content),
		namespace,
	)
	if err != nil {
		return err
	}

	snippet := content
	if len(snippet) > 200 {
		snippet = snippet[:200] + "..."
	}
	vector, err := e.embedder.ComputeEmbeddings(ctx, title+" "+snippet, false)
	if err != nil {
		log.Printf("[Engine] Failed to embed %s: %v", urlStr, err)
	}

	if err := e.qdrantClient.Upsert(ctx, "documents", urlStr, title, snippet, vector); err != nil {
		log.Printf("[Engine] Qdrant upsert failed for %s: %v", urlStr, err)
	}

	if title != "" {
		err = e.redisClient.ZAdd(ctx, "rarefactor:autocomplete", redis.Z{
			Score:  0,
			Member: strings.ToLower(title),
		}).Err()
		if err != nil {
			log.Printf("[Redis] Failed to add title to autocomplete: %v", err)
		}
	}

	return nil
}
