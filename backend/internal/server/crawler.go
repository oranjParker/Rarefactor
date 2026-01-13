package server

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/oranjParker/Rarefactor/generated/protos/v1"
	"github.com/oranjParker/Rarefactor/internal/crawler/engine"
	"github.com/oranjParker/Rarefactor/internal/database"
	"github.com/oranjParker/Rarefactor/internal/search"
	"github.com/redis/go-redis/v9"
)

type CrawlerServer struct {
	db  *pgxpool.Pool
	eng *engine.Engine
	qdb *database.QdrantClient
	emb *search.Embedder
}

func NewCrawlerServer(db *pgxpool.Pool, rdb *redis.Client, qdb *database.QdrantClient, emb *search.Embedder) *CrawlerServer {
	eng := engine.NewEngine(db, 25, 2*time.Second, rdb, qdb, emb)
	return &CrawlerServer{db: db, eng: eng}
}

func (c *CrawlerServer) Crawl(ctx context.Context, req *pb.CrawlRequest) (*pb.CrawlResponse, error) {
	log.Printf("[RPC] Received Crawl Trigger: %s", req.SeedUrl)

	go func() {
		// Enforce a hard timeout for background crawls to prevent infinite resource usage
		crawlCtx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()

		c.eng.Run(crawlCtx, req.SeedUrl, "production-crawl")
	}()

	return &pb.CrawlResponse{
		Status: "Crawl started in background",
	}, nil
}
