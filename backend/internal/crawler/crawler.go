package crawler

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/oranjParker/Rarefactor/generated/protos/v1"
	"github.com/oranjParker/Rarefactor/internal/database"
	"github.com/oranjParker/Rarefactor/internal/search"
	"github.com/redis/go-redis/v9"
)

type CrawlerServer struct {
	db        *pgxpool.Pool
	eng       EngineRunner
	qdb       *database.QdrantClient
	emb       *search.Embedder
	serverCtx context.Context
}

func NewCrawlerServer(serverCtx context.Context, db *pgxpool.Pool, rdb *redis.Client, qdb *database.QdrantClient, emb *search.Embedder) *CrawlerServer {
	// TODO: Make config options
	eng := NewEngine(db, 50, 2*time.Second, rdb, qdb, emb)

	return &CrawlerServer{
		db:        db,
		eng:       eng,
		qdb:       qdb,
		emb:       emb,
		serverCtx: serverCtx,
	}
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
		crawlCtx, cancel := context.WithTimeout(c.serverCtx, 2*time.Hour)
		defer cancel()

		c.eng.Run(crawlCtx, req.SeedUrl, "production-crawl", depth, mode)
	}()

	return &pb.CrawlResponse{
		Status: "Crawl started in background",
	}, nil
}
