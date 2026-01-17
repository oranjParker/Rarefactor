package crawler

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestCoordinator_Fairness(t *testing.T) {
	c := NewCoordinator(1 * time.Second)

	for i := 0; i < 5; i++ {
		c.AddURL(fmt.Sprintf("http://domain-a.com/page%d", i), 0, 10, "")
	}
	c.AddURL("http://domain-b.com/only-page", 0, 10, "")

	job1, _ := c.GetNextJob()
	if !strings.Contains(job1.URL, "domain-a.com") {
		t.Errorf("Expected domain-a, got %s", job1.URL)
	}

	job2, ok := c.GetNextJob()
	if !ok || !strings.Contains(job2.URL, "domain-b.com") {
		t.Errorf("Fairness failed: Expected domain-b to jump the queue, got %s", job2.URL)
	}
}

func TestCoordinator_Duplicates(t *testing.T) {
	c := NewCoordinator(1 * time.Second)
	url := "http://example.com"

	c.AddURL(url, 0, 10, "")
	c.AddURL(url, 0, 10, "")

	c.GetNextJob()
	_, ok := c.GetNextJob()

	if ok {
		t.Error("Duplicate URL was allowed into the queue")
	}
}

func TestCoordinator_State(t *testing.T) {
	c := NewCoordinator(10 * time.Millisecond)

	c.IncrementActiveWorkers()
	if c.ActiveWorkers() != 1 {
		t.Errorf("Expected 1 active worker, got %d", c.ActiveWorkers())
	}

	c.AddURL("http://example.com", 0, 2, "")
	if !c.HasWork() {
		t.Error("Expected coordinator to have work")
	}

	res := CrawlResult{
		URL:   "http://example.com",
		Depth: 0,
		Links: []string{"http://example.com/a"},
	}
	c.ProcessResult(res)

	if c.ActiveWorkers() != 0 {
		t.Errorf("Expected 0 active workers after ProcessResult, got %d", c.ActiveWorkers())
	}
}

func TestCoordinator_Timing(t *testing.T) {
	c := NewCoordinator(100 * time.Millisecond)
	c.AddURL("http://example.com", 0, 2, "")

	job, ok := c.GetNextJob()
	if !ok || job.URL != "http://example.com" {
		t.Fatal("Failed to get first job")
	}

	delay := c.TimeToNextJob()
	if delay <= 0 {
		t.Errorf("Expected positive delay due to politeness, got %v", delay)
	}

	time.Sleep(110 * time.Millisecond)
	delay = c.TimeToNextJob()
	if delay != 0 {
		t.Errorf("Expected 0 delay after waiting, got %v", delay)
	}
}
