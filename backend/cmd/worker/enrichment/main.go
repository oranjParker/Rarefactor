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
	"github.com/nats-io/nats.go"
	"github.com/oranjParker/Rarefactor/internal/core"
	"github.com/oranjParker/Rarefactor/internal/database"
	"github.com/oranjParker/Rarefactor/internal/llm_provider"
	"github.com/oranjParker/Rarefactor/internal/processor"
	"github.com/oranjParker/Rarefactor/internal/sink"
	"github.com/oranjParker/Rarefactor/internal/source"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	deps, err := setupWorkerDependencies(ctx)
	if err != nil {
		log.Fatalf("Infrastructure failure: %v", err)
	}
	defer deps.Nats.Close()
	defer deps.Postgres.Close()
	defer deps.Qdrant.Close()

	var llmProvider processor.LLMProvider
	geminiKey := os.Getenv("GEMINI_API_KEY")
	ollamaURL := os.Getenv("OLLAMA_URL")

	if geminiKey != "" {
		log.Println("[LLM] Using Gemini (Cloud Free Tier)")
		llmProvider, _ = llm_provider.NewGeminiProvider(ctx, geminiKey)
	} else if ollamaURL != "" {
		log.Println("[LLM] Using Ollama (Local/Zero-Cost)")
		llmProvider = llm_provider.NewOllamaProvider(ollamaURL, "mistral")
	} else {
		log.Println("[LLM] No LLM config found. Using Mock Provider for testing.")
		llmProvider = &llm_provider.MockProvider{}
	}

	enrichmentSrc := source.NewNatsSource(deps.Nats.JS, "crawl.enrichment", "enrichment-group")
	pgSink := sink.NewPostgresSink(deps.Postgres, 50, 5*time.Second)
	defer pgSink.Close()

	qdrantSink := sink.NewQdrantSink(deps.Qdrant, "documents")

	runner := core.NewGraphRunner("Rarefactor-V2", enrichmentSrc, 3)

	if err := runner.AddProcessor("start", processor.NewMetadataProcessor(llmProvider)); err != nil {
		log.Fatalf("Failed to add node 'metadata': %v", err)
	}

	if err := runner.AddHybrid("embedding", processor.NewEmbeddingProcessor(os.Getenv("EMBEDDING_URL")), qdrantSink); err != nil {
		log.Fatalf("Failed to add node 'embedding': %v", err)
	}

	if err := runner.AddSink("persist_pg", pgSink); err != nil {
		log.Fatalf("Failed to add node 'persist_pg': %v", err)
	}

	if err := runner.Connect("start", "embedding"); err != nil {
		log.Fatalf("Graph wiring failed: %v", err)
	}
	if err := runner.Connect("embedding", "persist_pg"); err != nil {
		log.Fatalf("Graph wiring failed: %v", err)
	}

	log.Println("[Graph] Enrichment Topology constructed. Starting engine...")
	if err := runner.Run(ctx); err != nil {
		log.Printf("Worker stopped: %v", err)
	}
}

type WorkerDependencies struct {
	Nats     *database.NatsConn
	Postgres *pgxpool.Pool
	Qdrant   *database.QdrantClient
}

func setupWorkerDependencies(ctx context.Context) (*WorkerDependencies, error) {
	deadline := time.Now().Add(120 * time.Second)

	var (
		pg  *pgxpool.Pool
		nt  *database.NatsConn
		qdb *database.QdrantClient
		err error
	)

	log.Println("[Init] Waiting for infrastructure (NATS, Postgres, Redis, Qdrant)...")

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if pg == nil {
			pg, err = database.NewPool(ctx)
			if err != nil {
				log.Printf("Postgres not ready: %v", err)
				goto retry
			}
		}

		if nt == nil {
			nt, err = database.NewNatsConnection()
			if err != nil {
				log.Printf("NATS not ready: %v", err)
				goto retry
			}

			_, err = nt.JS.AddStream(&nats.StreamConfig{
				Name:      "CRAWL_JOBS",
				Subjects:  []string{"crawl.>"},
				Retention: nats.WorkQueuePolicy,
				MaxMsgs:   1000000,
				MaxBytes:  10 * 1024 * 1024 * 1024,
				Discard:   nats.DiscardOld,
			})
			if err != nil {
				log.Printf("Stream setup failed: %v", err)
				nt.Close()
				nt = nil
				goto retry
			}
		}
		if qdb == nil {
			qdb, err = database.NewQdrantClient(ctx)
			if err != nil {
				log.Printf("Qdrant not ready: %v", err)
				goto retry
			}
			if err := qdb.EnsureCollection(ctx, "documents"); err != nil {
				log.Printf("Qdrant collection warning: %v", err)
			}
		}

		return &WorkerDependencies{
			Nats:     nt,
			Postgres: pg,
			Qdrant:   qdb,
		}, nil

	retry:
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("infrastructure initialization timed out after 120s")
}
