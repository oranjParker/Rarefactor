package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oranjParker/Rarefactor/internal/core"
	"github.com/oranjParker/Rarefactor/internal/database"
	"github.com/oranjParker/Rarefactor/internal/processor"
	"github.com/oranjParker/Rarefactor/internal/sink"
	"github.com/oranjParker/Rarefactor/internal/source"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	deps, err := setupWorkerDependencies(ctx)
	if err != nil {
		log.Fatalf("Infrastructure failure: %v", err)
	}
	defer deps.Nats.Close()
	defer deps.Redis.Close()
	defer deps.Postgres.Close()
	defer deps.Qdrant.Close()

	var llmProvider processor.LLMProvider
	geminiKey := os.Getenv("GEMINI_API_KEY")
	ollamaURL := os.Getenv("OLLAMA_URL")

	if geminiKey != "" {
		log.Println("[LLM] Using Gemini (Cloud Free Tier)")
		llmProvider, _ = processor.NewGeminiProvider(ctx, geminiKey)
	} else if ollamaURL != "" {
		log.Println("[LLM] Using Ollama (Local/Zero-Cost)")
		llmProvider = processor.NewOllamaProvider(ollamaURL, "llama3")
	} else {
		log.Println("[LLM] No LLM config found. Using Mock Provider for testing.")
		llmProvider = &processor.MockProvider{}
	}

	natsSrc := source.NewNatsSource(deps.Nats.JS, "crawl.jobs", "worker-group")
	pgSink := sink.NewPostgresSink(deps.Postgres, 50, 5*time.Second)
	defer pgSink.Close()

	smartCrawler := processor.NewSmartCrawlerProcessor()

	runner := core.NewPipelineRunner(natsSrc, pgSink, core.PipelineConfig{
		Concurrency: 5,
		Name:        "Rarefactor-V2-Worker",
	})

	runner.AddProcessor(processor.NewPolitenessProcessor(deps.Redis, "RarefactorBot/2.0", 2, 1000))
	runner.AddProcessor(smartCrawler)
	runner.AddProcessor(processor.NewDiscoveryProcessor(sink.NewNatsSink(deps.Nats.JS, "crawl.jobs")))
	runner.AddProcessor(processor.NewSecurityProcessor(false))
	runner.AddProcessor(processor.NewChunkerProcessor(1000, 200))
	runner.AddProcessor(processor.NewEnrichmentProcessor())
	runner.AddProcessor(processor.NewMetadataProcessor(llmProvider))
	runner.AddProcessor(processor.NewVectorForkProcessor(sink.NewNatsSink(deps.Nats.JS, "vector.jobs")))

	log.Println("Worker ready. Awaiting jobs...")
	if err := runner.Run(ctx); err != nil {
		log.Printf("Worker exited: %v", err)
	}
}

type WorkerDependencies struct {
	Nats     *database.NatsConn
	Redis    *redis.Client
	Postgres *pgxpool.Pool
	Qdrant   *database.QdrantClient
}

func setupWorkerDependencies(ctx context.Context) (*WorkerDependencies, error) {
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	pg, err := database.NewPool(initCtx)
	if err != nil {
		return nil, fmt.Errorf("postgres init: %w", err)
	}

	rdb, err := database.NewRedisClient(initCtx)
	if err != nil {
		pg.Close()
		return nil, fmt.Errorf("redis init: %w", err)
	}

	nt, err := database.NewNatsConnection()
	if err != nil {
		pg.Close()
		rdb.Close()
		return nil, fmt.Errorf("nats init: %w", err)
	}

	qdb, err := database.NewQdrantClient(initCtx)
	if err != nil {
		pg.Close()
		rdb.Close()
		nt.Close()
		return nil, fmt.Errorf("qdrant init: %w", err)
	}

	if err := qdb.EnsureCollection(initCtx, "documents"); err != nil {
		log.Printf("Warning: Qdrant collection setup: %v", err)
	}

	return &WorkerDependencies{
		Nats:     nt,
		Redis:    rdb,
		Postgres: pg,
		Qdrant:   qdb,
	}, nil
}
