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
