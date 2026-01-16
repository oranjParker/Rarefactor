package crawler

import (
	"container/heap"
	"math"
	"net/url"
	"time"

	"github.com/oranjParker/Rarefactor/internal/utils"
)

type CrawlResult struct {
	URL           string
	Depth         int
	MaxDepth      int
	AllowedDomain string
	Title         string
	Content       string
	Links         []string
	Err           error
}

type URLJob struct {
	URL           string
	Depth         int
	MaxDepth      int
	AllowedDomain string
}

type DomainRecord struct {
	Domain    string
	LastCrawl time.Time
	PageCount int
}

type DomainHeap []DomainRecord

func (h DomainHeap) Len() int { return len(h) }
func (h DomainHeap) Less(i, j int) bool {
	penaltyI := math.Log1p(float64(h[i].PageCount)) * 10
	penaltyJ := math.Log1p(float64(h[j].PageCount)) * 10

	weightI := h[i].LastCrawl.Add(time.Duration(penaltyI) * time.Second)
	weightJ := h[j].LastCrawl.Add(time.Duration(penaltyJ) * time.Second)

	return weightI.Before(weightJ)
}
func (h DomainHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *DomainHeap) Push(x any)   { *h = append(*h, x.(DomainRecord)) }

func (h *DomainHeap) Pop() any {
	old := *h
	n := len(old)
	val := old[n-1]
	*h = old[0 : n-1]
	return val
}

type Coordinator struct {
	JobsChan          chan URLJob
	ResultsChan       chan CrawlResult
	domainQueue       map[string][]URLJob
	history           *DomainHeap
	domainCounts      map[string]int
	visited           map[string]struct{}
	politeness        time.Duration
	activeWorkers     int
	maxPagesPerDomain int
}

func NewCoordinator(politeness time.Duration) *Coordinator {
	h := &DomainHeap{}
	heap.Init(h)
	return &Coordinator{
		JobsChan:          make(chan URLJob),
		ResultsChan:       make(chan CrawlResult),
		domainQueue:       make(map[string][]URLJob),
		history:           h,
		domainCounts:      make(map[string]int),
		visited:           make(map[string]struct{}),
		politeness:        politeness,
		maxPagesPerDomain: 1000,
	}
}

func (c *Coordinator) AddURL(rawURL string, depth, maxDepth int, allowedDomain string) {
	if depth > maxDepth {
		return
	}
	if _, ok := c.visited[rawURL]; ok {
		return
	}

	baseDomain, err := utils.GetBaseDomain(rawURL)
	if err != nil {
		return
	}

	if allowedDomain != "" && baseDomain != allowedDomain {
		return
	}

	if c.domainCounts[baseDomain] >= c.maxPagesPerDomain {
		return
	}
	c.visited[rawURL] = struct{}{}

	u, _ := url.Parse(rawURL)
	host := u.Host

	if _, ok := c.domainQueue[host]; !ok {
		c.domainQueue[host] = []URLJob{}
		record := &DomainRecord{
			Domain:    host,
			LastCrawl: time.Unix(0, 0),
			PageCount: 0,
		}
		heap.Push(c.history, *record)
	}

	c.domainQueue[host] = append(c.domainQueue[host], URLJob{URL: rawURL, Depth: depth, MaxDepth: maxDepth, AllowedDomain: allowedDomain})
}

func (c *Coordinator) ProcessResult(res CrawlResult) {
	c.activeWorkers--
	if res.Err != nil {
		return
	}
	for _, link := range res.Links {
		c.AddURL(link, res.Depth+1, res.MaxDepth, res.AllowedDomain)
	}
}

func (c *Coordinator) IncrementActiveWorkers() {
	c.activeWorkers++
}

func (c *Coordinator) HasWork() bool {
	if c.activeWorkers > 0 {
		return true
	}
	for _, q := range c.domainQueue {
		if len(q) > 0 {
			return true
		}
	}
	return false
}

func (c *Coordinator) GetNextJob() (URLJob, bool) {
	if c.history.Len() == 0 {
		return URLJob{}, false
	}
	nextRecord := (*c.history)[0]

	if time.Since(nextRecord.LastCrawl) < c.politeness {
		return URLJob{}, false
	}

	record := heap.Pop(c.history).(DomainRecord)

	baseDomain, _ := utils.GetBaseDomain("http://" + record.Domain)
	if c.domainCounts[baseDomain] >= c.maxPagesPerDomain {
		delete(c.domainQueue, record.Domain)
		return c.GetNextJob()
	}

	queue := c.domainQueue[record.Domain]
	if len(queue) == 0 {
		return c.GetNextJob()
	}

	nextJob := queue[0]
	c.domainQueue[record.Domain] = queue[1:]

	record.LastCrawl = time.Now()
	record.PageCount++
	c.domainCounts[baseDomain]++

	heap.Push(c.history, record)
	return nextJob, true
}

func (c *Coordinator) TimeToNextJob() time.Duration {
	if c.history.Len() == 0 {
		return 0
	}
	nextRecord := (*c.history)[0]
	remaining := c.politeness - time.Since(nextRecord.LastCrawl)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (c *Coordinator) ActiveWorkers() int {
	return c.activeWorkers
}
