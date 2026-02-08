package sink

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oranjParker/Rarefactor/internal/core"
)

const (
	DefaultBatchSize    = 50
	DefaultFlushTimeout = 5 * time.Second
)

type PostgresSink struct {
	db           *pgxpool.Pool
	batchSize    int
	flushTimeout time.Duration

	buffer []core.Document[string]
	mu     sync.Mutex

	flushChan chan struct{}
	closeChan chan struct{}
	wg        sync.WaitGroup
}

func NewPostgresSink(db *pgxpool.Pool, batchSize int, timeout time.Duration) *PostgresSink {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	if timeout <= 0 {
		timeout = DefaultFlushTimeout
	}

	s := &PostgresSink{
		db:           db,
		batchSize:    batchSize,
		flushTimeout: timeout,
		buffer:       make([]core.Document[string], 0, batchSize),
		flushChan:    make(chan struct{}, 1),
		closeChan:    make(chan struct{}),
	}
	s.wg.Add(1)
	go s.worker()

	return s
}

func (s *PostgresSink) Write(ctx context.Context, doc *core.Document[string]) error {
	s.mu.Lock()
	s.buffer = append(s.buffer, *doc)
	shouldFlush := len(s.buffer) >= s.batchSize
	s.mu.Unlock()

	if shouldFlush {
		select {
		case s.flushChan <- struct{}{}:
		default:
		}
	}
	return nil
}

func (s *PostgresSink) worker() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.flushTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-s.closeChan:
			s.flush(context.Background())
			return
		case <-s.flushChan:
			s.flush(context.Background())
		case <-ticker.C:
			s.flush(context.Background())
		}
	}
}

func (s *PostgresSink) flush(ctx context.Context) {
	s.mu.Lock()
	if len(s.buffer) == 0 {
		s.mu.Unlock()
		return
	}
	items := s.buffer
	s.buffer = make([]core.Document[string], 0, s.batchSize)
	s.mu.Unlock()

	batch := &pgx.Batch{}
	query := `
		INSERT INTO documents (url, title, content, created_at, namespace) 
		VALUES ($1, $2, $3, $4, 'default')
		ON CONFLICT (url) DO UPDATE SET 
		    title = EXCLUDED.title, 
		    content = EXCLUDED.content,
		    crawled_at = NOW()
	`

	for _, doc := range items {
		batch.Queue(query, doc.ID, doc.Metadata["title"], doc.Content, doc.CreatedAt)
	}

	br := s.db.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(items); i++ {
		if _, err := br.Exec(); err != nil {
			log.Printf("[PostgresSink] Batch exec error: %v\n", err)
		}
	}
}

func (s *PostgresSink) Close() error {
	close(s.closeChan)
	s.wg.Wait()
	return nil
}
