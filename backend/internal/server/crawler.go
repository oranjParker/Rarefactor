package server

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/oranjParker/Rarefactor/generated/protos/v1"
	"github.com/oranjParker/Rarefactor/internal/crawler"
	"github.com/oranjParker/Rarefactor/internal/database"
	"github.com/oranjParker/Rarefactor/internal/search"
	"github.com/redis/go-redis/v9"
)

type CrawlerServer struct {
	db  *pgxpool.Pool
	eng crawler.EngineRunner
	qdb *database.QdrantClient
	emb *search.Embedder
}

func NewCrawlerServer(db *pgxpool.Pool, rdb *redis.Client, qdb *database.QdrantClient, emb *search.Embedder) *CrawlerServer {
	eng := crawler.NewEngine(db, 50, 2*time.Second, rdb, qdb, emb)
	return &CrawlerServer{db: db, eng: eng, qdb: qdb, emb: emb}
}

func (c *CrawlerServer) Crawl(ctx context.Context, req *pb.CrawlRequest) (*pb.CrawlResponse, error) {
	log.Printf("[RPC] Received Crawl Trigger: %s (mode: %s, max_depth: %d)", req.SeedUrl, req.CrawlMode, req.MaxDepth)

	mode := req.CrawlMode
	if mode == "" {
		mode = "broad"
	}
	depth := int(req.MaxDepth)
	if depth == 0 {
		depth = 2
	}

	go func() {
		// Enforce a hard timeout for background crawls to prevent infinite resource usage
		crawlCtx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()

		c.eng.Run(crawlCtx, req.SeedUrl, "production-crawl", depth, mode)
	}()

	return &pb.CrawlResponse{
		Status: "Crawl started in background",
	}, nil
}
