package processor

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"time"

	"github.com/jimsmart/grobotstxt"
	"github.com/oranjParker/Rarefactor/internal/core"
	"github.com/oranjParker/Rarefactor/internal/utils"
	"github.com/redis/go-redis/v9"
)

const (
	RobotsTTL     = 24 * time.Hour
	VisitedPrefix = "visited:"
	CountKey      = "crawl_counts"
)

type PolitenessProcessor struct {
	Redis             *redis.Client
	UserAgent         string
	httpClient        *http.Client
	MaxDepth          int
	MaxPagesPerDomain int
}

func NewPolitenessProcessor(rdb *redis.Client, ua string, maxDepth, maxPages int) *PolitenessProcessor {
	return &PolitenessProcessor{
		Redis:             rdb,
		UserAgent:         ua,
		MaxDepth:          maxDepth,
		MaxPagesPerDomain: maxPages,
		httpClient: utils.NewSafeHTTPClient(utils.ClientConfig{
			Timeout: 5 * time.Second,
		}),
	}
}

func (p *PolitenessProcessor) Process(ctx context.Context, doc *core.Document[string]) ([]*core.Document[string], error) {
	u, err := url.Parse(doc.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	if doc.Depth > p.MaxDepth {
		return nil, fmt.Errorf("max depth %d exceeded", p.MaxDepth)
	}

	domain, _ := utils.GetBaseDomain(doc.ID)

	visitedKey := VisitedPrefix + doc.ID
	isNew, err := p.Redis.SetNX(ctx, visitedKey, "1", 30*24*time.Hour).Result()
	if err != nil {
		return nil, fmt.Errorf("redis visited check failed: %w", err)
	}
	if !isNew {
		return nil, nil
	}

	robotsData, err := p.getRobotsData(ctx, u)
	if err != nil {
		fmt.Printf("[Politeness] robots.txt warning for %s: %v\n", u.Host, err)
	} else if robotsData != "" {
		if !grobotstxt.AgentAllowed(robotsData, p.UserAgent, u.Path) {
			return nil, core.ErrRobotsDisallowed
		}
	}

	script := `
		local current = tonumber(redis.call("HGET", KEYS[1], ARGV[1]) or "0")
		if current >= tonumber(ARGV[2]) then
			return -1
		end
		return redis.call("HINCRBY", KEYS[1], ARGV[1], 1)
	`
	res, err := p.Redis.Eval(ctx, script, []string{CountKey}, domain, p.MaxPagesPerDomain).Int64()
	if err != nil {
		p.Redis.Del(ctx, visitedKey)
		return nil, err
	}
	if res == -1 {
		return nil, core.ErrQuotaExceeded
	}

	rollback := func() {
		p.Redis.HIncrBy(ctx, CountKey, domain, -1)
		p.Redis.Del(ctx, visitedKey)
	}

	if res > 1 {
		penaltySeconds := math.Log2(float64(res))
		elapsed := time.Since(doc.CreatedAt).Seconds()

		if elapsed < penaltySeconds {
			rollback()
			return nil, fmt.Errorf("%w: wait %.2fs", core.ErrDelayRequired, penaltySeconds-elapsed)
		}
	}

	return []*core.Document[string]{doc}, nil
}

func (p *PolitenessProcessor) getRobotsData(ctx context.Context, u *url.URL) (string, error) {
	robotsKey := fmt.Sprintf("robots:%s", u.Host)

	data, err := p.Redis.Get(ctx, robotsKey).Result()
	if err == nil {
		return data, nil
	}
	if err != redis.Nil {
		return "", err
	}

	robotsURL := fmt.Sprintf("%s://%s/robots.txt", u.Scheme, u.Host)
	req, err := http.NewRequestWithContext(ctx, "GET", robotsURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", p.UserAgent)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		p.Redis.Set(ctx, robotsKey, "", RobotsTTL)
		return "", nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", err
	}

	robotsContent := string(body)
	p.Redis.Set(ctx, robotsKey, robotsContent, RobotsTTL)
	return robotsContent, nil
}
