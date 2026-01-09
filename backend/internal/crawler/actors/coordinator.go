package actors

import (
	"container/heap"
	"net/url"
	"time"
)

type CrawlResult struct {
	URL     string
	Title   string
	Content string
	Links   []string
	Err     error
}

type DomainRecord struct {
	Domain    string
	LastCrawl time.Time
}
type DomainHeap []DomainRecord

func (h DomainHeap) Len() int           { return len(h) }
func (h DomainHeap) Less(i, j int) bool { return h[i].LastCrawl.Before(h[j].LastCrawl) }
func (h DomainHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *DomainHeap) Push(x any)        { *h = append(*h, x.(DomainRecord)) }

func (h *DomainHeap) Pop() any {
	old := *h
	n := len(old)
	val := old[n-1]
	*h = old[0 : n-1]
	return val
}

type Coordinator struct {
	JobsChan      chan string
	ResultsChan   chan CrawlResult
	domainQueue   map[string][]string
	history       *DomainHeap
	visited       map[string]struct{}
	politeness    time.Duration
	activeWorkers int
}

func NewCoordinator(politeness time.Duration) *Coordinator {
	h := &DomainHeap{}
	heap.Init(h)
	return &Coordinator{
		JobsChan:    make(chan string),
		ResultsChan: make(chan CrawlResult),
		domainQueue: make(map[string][]string),
		history:     h,
		visited:     make(map[string]struct{}),
		politeness:  politeness,
	}
}

func (c *Coordinator) AddURL(rawURL string) {
	if _, ok := c.visited[rawURL]; ok {
		return
	}
	c.visited[rawURL] = struct{}{}

	u, err := url.Parse(rawURL)
	if err != nil {
		return
	}
	domain := u.Host

	if _, ok := c.domainQueue[domain]; !ok {
		c.domainQueue[domain] = []string{}
		heap.Push(c.history, DomainRecord{Domain: domain, LastCrawl: time.Unix(0, 0)})
	}
	c.domainQueue[domain] = append(c.domainQueue[domain], rawURL)
}

func (c *Coordinator) ProcessResult(res CrawlResult) {
	c.activeWorkers--
	if res.Err != nil {
		return
	}
	for _, link := range res.Links {
		c.AddURL(link)
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

func (c *Coordinator) GetNextJob() (string, bool) {
	if c.history.Len() == 0 {
		return "", false
	}
	nextRecord := (*c.history)[0]

	if time.Since(nextRecord.LastCrawl) < c.politeness {
		return "", false
	}

	record := heap.Pop(c.history).(DomainRecord)

	queue := c.domainQueue[record.Domain]
	if len(queue) == 0 {
		return c.GetNextJob()
	}

	url := queue[0]
	c.domainQueue[record.Domain] = queue[1:]

	record.LastCrawl = time.Now()
	heap.Push(c.history, record)

	return url, true
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
