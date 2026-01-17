package crawler

import (
	"container/heap"
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
	c.AddURL("http://example.com/1", 0, 2, "")
	c.AddURL("http://example.com/2", 0, 2, "")

	job, ok := c.GetNextJob()
	if !ok || job.URL != "http://example.com/1" {
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

func TestCoordinator_MaxPagesPerDomain(t *testing.T) {
	c := NewCoordinator(0)
	c.maxPagesPerDomain = 2

	c.AddURL("http://example.com/1", 0, 10, "")
	c.AddURL("http://example.com/2", 0, 10, "")
	c.AddURL("http://example.com/3", 0, 10, "") // Should be dropped in AddURL or GetNextJob

	_, ok1 := c.GetNextJob()
	_, ok2 := c.GetNextJob()
	_, ok3 := c.GetNextJob()

	if !ok1 || !ok2 {
		t.Fatal("Expected first two jobs to be allowed")
	}
	if ok3 {
		t.Error("Expected third job to be rejected due to maxPagesPerDomain")
	}

	// Test AddURL rejecting immediately if limit already reached
	c.baseDomainCounts["example.com"] = 2
	c.AddURL("http://example.com/4", 0, 10, "")
	for _, job := range c.domains["example.com"].Queue {
		if job.URL == "http://example.com/4" {
			t.Error("AddURL should have rejected URL 4 due to limit")
		}
	}
}

func TestCoordinator_AddURL_Edges(t *testing.T) {
	c := NewCoordinator(0)

	// Depth > MaxDepth
	c.AddURL("http://example.com/too-deep", 5, 2, "")
	if len(c.visited) > 0 {
		t.Error("Should not have added deep URL")
	}

	// Invalid URL (GetBaseDomain failure)
	c.AddURL(":", 0, 10, "")
	if len(c.visited) > 0 {
		t.Error("Should not have added invalid URL")
	}

	// No host
	c.AddURL("mailto:test@example.com", 0, 10, "")
	if len(c.domains) > 0 {
		t.Error("Should not have created domain state for mailto")
	}

	// Allowed domain restriction
	c.AddURL("http://other.com", 0, 10, "example.com")
	if _, ok := c.domains["other.com"]; ok {
		t.Error("Should not have added URL from different domain")
	}
}

func TestCoordinator_GetNextJob_Edges(t *testing.T) {
	c := NewCoordinator(0)

	// No jobs
	_, ok := c.GetNextJob()
	if ok {
		t.Error("GetNextJob should return false when empty")
	}

	// Job with empty queue (should call itself recursively)
	state := &DomainState{Host: "empty.com", Queue: []URLJob{}}
	c.domains["empty.com"] = state
	heap.Push(c.history, state)
	state.InHeap = true

	_, ok = c.GetNextJob()
	if ok {
		t.Error("GetNextJob should return false for empty queue")
	}
}

func TestCoordinator_HasWork(t *testing.T) {
	c := NewCoordinator(0)
	if c.HasWork() {
		t.Error("Should not have work initially")
	}

	c.AddURL("http://example.com", 0, 10, "")
	if !c.HasWork() {
		t.Error("Should have work after adding URL")
	}

	c.GetNextJob()
	if c.HasWork() {
		// History is empty now
		t.Error("Should not have work after pulling only job")
	}

	c.IncrementActiveWorkers()
	if !c.HasWork() {
		t.Error("Should have work if active workers > 0")
	}
}

func TestCoordinator_ProcessResult(t *testing.T) {
	c := NewCoordinator(0)
	c.IncrementActiveWorkers()

	res := CrawlResult{
		URL:   "http://example.com",
		Depth: 0,
		Links: []string{"http://example.com/1"},
		Err:   fmt.Errorf("some error"),
	}
	c.ProcessResult(res)
	if c.ActiveWorkers() != 0 {
		t.Error("Worker should be decremented even on error")
	}
	if len(c.visited) > 1 { // Only example.com (from some previous state? no, this is new C)
		// Wait, visited is not set in NewCoordinator except as empty map.
		// example.com wasn't added to visited here.
		if _, ok := c.visited["http://example.com/1"]; ok {
			t.Error("Should not have added link on error")
		}
	}
}
