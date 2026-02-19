package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/oranjParker/Rarefactor/generated/protos/v1"
	"github.com/oranjParker/Rarefactor/internal/core"
	"github.com/oranjParker/Rarefactor/internal/database"
)

type CrawlerService struct {
	pb.UnimplementedCrawlerServiceServer
	db   *pgxpool.Pool
	nats *database.NatsConn
}

func NewCrawlerService(db *pgxpool.Pool, nats *database.NatsConn) *CrawlerService {
	return &CrawlerService{
		db:   db,
		nats: nats,
	}
}

func (s *CrawlerService) Crawl(ctx context.Context, req *pb.CrawlRequest) (*pb.CrawlResponse, error) {
	if req.SeedUrl == "" {
		return nil, fmt.Errorf("seed_url is required")
	}

	jobID := uuid.New().String()

	query := `
		INSERT INTO crawl_jobs (id, seed_url, max_depth, crawl_mode, namespace, status, created_at)
		VALUES ($1, $2, $3, $4, $5, 'PENDING', NOW())
	`
	_, err := s.db.Exec(ctx, query, jobID, req.SeedUrl, req.MaxDepth, req.CrawlMode, "test")
	if err != nil {
		log.Printf("[API] Failed to persist job: %v", err)
		return nil, fmt.Errorf("internal database error")
	}

	seedDoc := &core.Document[string]{
		ID:        req.SeedUrl,
		Source:    "api_trigger",
		Depth:     0,
		CreatedAt: time.Now(),
		Metadata: map[string]any{
			"job_id":    jobID,
			"max_depth": req.MaxDepth,
			"mode":      req.CrawlMode,
		},
	}

	payload, err := json.Marshal(seedDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job payload: %w", err)
	}

	if _, err := s.nats.JS.Publish("crawl.jobs", payload); err != nil {
		_, _ = s.db.Exec(ctx, "UPDATE crawl_jobs SET status = 'FAILED' WHERE id = $1", jobID)
		return nil, fmt.Errorf("failed to queue job: %w", err)
	}

	log.Printf("[API] Job Queued: %s -> %s", jobID, req.SeedUrl)

	return &pb.CrawlResponse{
		JobId:  jobID,
		Status: "QUEUED",
	}, nil
}

func (s *CrawlerService) CancelJob(ctx context.Context, req *pb.CancelJobRequest) (*pb.CancelJobResponse, error) {
	_, err := s.db.Exec(ctx, "UPDATE crawl_jobs SET status = 'CANCELLED' WHERE id = $1", req.JobId)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel job: %w", err)
	}

	return &pb.CancelJobResponse{Status: "CANCELLED_SIGNAL_SENT"}, nil
}
