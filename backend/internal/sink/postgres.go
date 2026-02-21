package sink

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oranjParker/Rarefactor/internal/core"
)

const (
	DefaultBatchSize    = 20
	DefaultFlushTimeout = 10 * time.Second
)

type PostgresSink struct {
	db            *pgxpool.Pool
	batchSize     int
	flushInterval time.Duration

	buffer []*core.Document[string]
	mu     sync.Mutex

	closeChan chan struct{}
	wg        sync.WaitGroup
}

func NewPostgresSink(db *pgxpool.Pool, batchSize int, interval time.Duration) *PostgresSink {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	if interval <= 0 {
		interval = DefaultFlushTimeout
	}

	s := &PostgresSink{
		db:            db,
		batchSize:     batchSize,
		flushInterval: interval,
		buffer:        make([]*core.Document[string], 0, batchSize),
		closeChan:     make(chan struct{}),
	}

	s.wg.Add(1)
	go s.runFlusher()

	return s
}

func (s *PostgresSink) Write(ctx context.Context, doc *core.Document[string]) error {
	s.mu.Lock()
	s.buffer = append(s.buffer, doc)
	shouldFlush := len(s.buffer) >= s.batchSize
	s.mu.Unlock()

	if shouldFlush {
		return s.flush(ctx)
	}
	return nil
}

func (s *PostgresSink) flush(ctx context.Context) error {
	s.mu.Lock()
	if len(s.buffer) == 0 {
		s.mu.Unlock()
		return nil
	}

	pending := s.buffer
	s.buffer = make([]*core.Document[string], 0, s.batchSize)
	s.mu.Unlock()

	return s.executeBatch(ctx, pending)
}

func (s *PostgresSink) executeBatch(ctx context.Context, items []*core.Document[string]) error {
	batch := &pgx.Batch{}
	log.Printf("[PostgresSink] Executing batch of %d items\n", len(items))
	query := `
		INSERT INTO documents (
			id, 
			parent_id, 
			namespace, 
			domain, 
			source, 
			content, 
			cleaned_content, 
			title, 
			summary, 
			content_hash, 
			metadata, 
			crawled_at, 
			last_seen_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		ON CONFLICT (id) DO UPDATE SET
			content = EXCLUDED.content,
			cleaned_content = EXCLUDED.cleaned_content,
			title = EXCLUDED.title,
			summary = EXCLUDED.summary,
			content_hash = EXCLUDED.content_hash,
			metadata = EXCLUDED.metadata,
			last_seen_at = NOW()
	`

	jobQuery := `
		UPDATE crawl_jobs 
		SET pages_crawled = pages_crawled + $2, updated_at = NOW() 
		WHERE id = $1
	`

	jobUpdates := make(map[string]int)

	for _, doc := range items {
		domain := extractDomain(doc.ID)

		namespace, _ := doc.Metadata["namespace"].(string)
		if namespace == "" {
			namespace = "default"
		}
		title, _ := doc.Metadata["title"].(string)
		summary, _ := doc.Metadata["summary"].(string)
		jobID, _ := doc.Metadata["job_id"].(string)
		contentHash := generateHash(doc.Content)

		batch.Queue(query,
			doc.ID,
			doc.ParentID,
			namespace,
			domain,
			doc.Source,
			doc.Content,
			doc.CleanedContent,
			title,
			summary,
			contentHash,
			doc.Metadata,
			doc.CreatedAt,
		)

		if jobID != "" {
			var chunkIdx int
			if val, ok := doc.Metadata["chunk_index"]; ok {
				switch v := val.(type) {
				case int:
					chunkIdx = v
				case float64:
					chunkIdx = int(v)
				}
			}

			isChunk := doc.ParentID != ""
			if !isChunk || (isChunk && chunkIdx == 0) {
				jobUpdates[jobID]++
			}
		}
	}

	for id, delta := range jobUpdates {
		batch.Queue(jobQuery, id, delta)
	}

	br := s.db.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(items); i++ {
		_, err := br.Exec()
		if err != nil {
			log.Printf("[PostgresSink] Batch exec error for item %s: %v\n", items[i].ID, err)
		} else {
			if items[i].Ack != nil {
				items[i].Ack()
			}
		}
	}

	for i := 0; i < len(jobUpdates); i++ {
		if _, err := br.Exec(); err != nil {
			log.Printf("[PostgresSink] Job Stats update error: %v", err)
		}
	}

	return nil
}

func (s *PostgresSink) runFlusher() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.flush(context.Background()); err != nil {
				log.Printf("[PostgresSink] Scheduled flush failed: %v", err)
			}
		case <-s.closeChan:
			return
		}
	}
}

func (s *PostgresSink) Close() error {
	close(s.closeChan)
	s.wg.Wait()
	return s.flush(context.Background())
}

func generateHash(content string) string {
	if content == "" {
		return ""
	}
	h := sha256.New()
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))
}

func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "unknown"
	}
	return strings.TrimPrefix(u.Hostname(), "www.")
}
