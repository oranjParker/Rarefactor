package crawler

import (
	"container/heap"
	"math"
	"net/url"
	"time"

	"github.com/oranjParker/Rarefactor/internal/utils"
)

type URLJob struct {
	URL           string
	Depth         int
	MaxDepth      int
	AllowedDomain string
}

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

type DomainState struct {
	Host       string
	Queue      []URLJob
	LastCrawl  time.Time
	PageCount  int
	InHeap     bool
	BaseDomain string
}

type DomainHeap []*DomainState

func (h DomainHeap) Len() int { return len(h) }
func (h DomainHeap) Less(i, j int) bool {
	penaltyI := math.Log1p(float64(h[i].PageCount)) * 10
	penaltyJ := math.Log1p(float64(h[j].PageCount)) * 10

	weightI := h[i].LastCrawl.Add(time.Duration(penaltyI) * time.Second)
	weightJ := h[j].LastCrawl.Add(time.Duration(penaltyJ) * time.Second)

	return weightI.Before(weightJ)
}
func (h DomainHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}
func (h *DomainHeap) Push(x any) { *h = append(*h, x.(*DomainState)) }
func (h *DomainHeap) Pop() any {
	old := *h
	n := len(old)
	val := old[n-1]
	*h = old[0 : n-1]
	return val
}

type Coordinator struct {
	domains map[string]*DomainState
	history *DomainHeap

	baseDomainCounts map[string]int
	visited          map[string]struct{}

	politeness        time.Duration
	maxPagesPerDomain int

	activeWorkers int

	JobsChan    chan URLJob
	ResultsChan chan CrawlResult
}

func NewCoordinator(politeness time.Duration) *Coordinator {
	h := &DomainHeap{}
	heap.Init(h)
	return &Coordinator{
		domains:           make(map[string]*DomainState),
		history:           h,
		baseDomainCounts:  make(map[string]int),
		visited:           make(map[string]struct{}),
		politeness:        politeness,
		maxPagesPerDomain: 1000,
		JobsChan:          make(chan URLJob, 100),
		ResultsChan:       make(chan CrawlResult, 100),
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

	if c.baseDomainCounts[baseDomain] >= c.maxPagesPerDomain {
		return
	}

	if allowedDomain != "" && baseDomain != allowedDomain {
		return
	}

	c.visited[rawURL] = struct{}{}

	u, _ := url.Parse(rawURL)
	host := u.Host
	if host == "" {
		return
	}

	state, exists := c.domains[host]
	if !exists {
		state = &DomainState{
			Host:       host,
			Queue:      make([]URLJob, 0),
			BaseDomain: baseDomain,
		}
		c.domains[host] = state
	}

	job := URLJob{
		URL:           rawURL,
		Depth:         depth,
		MaxDepth:      maxDepth,
		AllowedDomain: allowedDomain,
	}

	state.Queue = append(state.Queue, job)

	if !state.InHeap {
		heap.Push(c.history, state)
		state.InHeap = true
	}
}

func (c *Coordinator) GetNextJob() (URLJob, bool) {
	if c.history.Len() == 0 {
		return URLJob{}, false
	}

	state := (*c.history)[0]

	if time.Since(state.LastCrawl) < c.politeness {
		return URLJob{}, false
	}

	heap.Pop(c.history)
	state.InHeap = false

	if c.baseDomainCounts[state.BaseDomain] >= c.maxPagesPerDomain {
		state.Queue = nil
		return c.GetNextJob()
	}

	if len(state.Queue) == 0 {
		return c.GetNextJob()
	}

	job := state.Queue[0]
	state.Queue = state.Queue[1:]

	state.LastCrawl = time.Now()
	state.PageCount++
	c.baseDomainCounts[state.BaseDomain]++

	if len(state.Queue) > 0 {
		heap.Push(c.history, state)
		state.InHeap = true
	}

	return job, true
}

func (c *Coordinator) TimeToNextJob() time.Duration {
	if c.history.Len() == 0 {
		return 0
	}

	state := (*c.history)[0]
	timeSinceLast := time.Since(state.LastCrawl)

	if timeSinceLast >= c.politeness {
		return 0
	}

	return c.politeness - timeSinceLast
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

func (c *Coordinator) IncrementActiveWorkers() { c.activeWorkers++ }
func (c *Coordinator) ActiveWorkers() int      { return c.activeWorkers }
func (c *Coordinator) HasWork() bool {
	if c.activeWorkers > 0 {
		return true
	}
	return c.history.Len() > 0
}
