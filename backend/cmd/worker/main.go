package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	pb "github.com/oranjParker/Rarefactor/generated/protos/v1"
	"github.com/oranjParker/Rarefactor/internal/api/crawler"
	"github.com/oranjParker/Rarefactor/internal/core"
	"github.com/oranjParker/Rarefactor/internal/database"
	"github.com/oranjParker/Rarefactor/internal/processor"
	"github.com/oranjParker/Rarefactor/internal/sink"
	"github.com/oranjParker/Rarefactor/internal/source"
	"github.com/redis/go-redis/v9"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const GRPC_PORT = ":50051"

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
		llmProvider = processor.NewOllamaProvider(ollamaURL, "mistral")
	} else {
		log.Println("[LLM] No LLM config found. Using Mock Provider for testing.")
		llmProvider = &processor.MockProvider{}
	}

	// =========================================================================
	// CONTROL PLANE: Start gRPC Server (Simulated API Pod)
	// =========================================================================
	go func() {
		listener, err := net.Listen("tcp", GRPC_PORT)
		if err != nil {
			log.Fatalf("[Control Plane] Failed to listen on %s: %v", GRPC_PORT, err)
		}

		grpcServer := grpc.NewServer()
		crawlerService := crawler.NewCrawlerService(deps.Postgres, deps.Nats)
		pb.RegisterCrawlerServiceServer(grpcServer, crawlerService)

		reflection.Register(grpcServer)

		log.Printf("[Control Plane] gRPC API listening on %s", GRPC_PORT)
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("[Control Plane] gRPC server error: %v", err)
		}
	}()

	// =========================================================================
	// DATA PLANE: Start Graph Runner (Simulated Worker Pods)
	// =========================================================================
	natsSrc := source.NewNatsSource(deps.Nats.JS, "crawl.jobs", "worker-group")
	pgSink := sink.NewPostgresSink(deps.Postgres, 50, 5*time.Second)
	defer pgSink.Close()

	linkSink := sink.NewNatsSink(deps.Nats.JS, "crawl.jobs")
	qdrantSink := sink.NewQdrantSink(deps.Qdrant, "documents")

	runner := core.NewGraphRunner("Rarefactor-V2", natsSrc, 3)

	if err := runner.AddProcessor("start", processor.NewPolitenessProcessor(deps.Redis, "RarefactorBot/2.0", 3, 1000)); err != nil {
		log.Fatalf("Failed to add node 'start': %v", err)
	}
	if err := runner.AddProcessor("crawler", processor.NewSmartCrawlerProcessor()); err != nil {
		log.Fatalf("Failed to add node 'crawler': %v", err)
	}
	if err := runner.AddHybrid("discovery", processor.NewDiscoveryProcessor(), linkSink); err != nil {
		log.Fatalf("Failed to add node 'discovery': %v", err)
	}
	if err := runner.AddProcessor("security", processor.NewSecurityProcessor(false)); err != nil { // false = don't fail, just flag
		log.Fatalf("Failed to add node 'security': %v", err)
	}
	if err := runner.AddProcessor("chunker", processor.NewChunkerProcessor(4000, 400)); err != nil {
		log.Fatalf("Failed to add node 'chunker': %v", err)
	}
	if err := runner.AddProcessor("enrichment", processor.NewEnrichmentProcessor()); err != nil {
		log.Fatalf("Failed to add node 'enrichment': %v", err)
	}
	if err := runner.AddProcessor("metadata", processor.NewMetadataProcessor(llmProvider)); err != nil {
		log.Fatalf("Failed to add node 'metadata': %v", err)
	}
	if err := runner.AddSink("persist_pg", pgSink); err != nil {
		log.Fatalf("Failed to add node 'persist_pg': %v", err)
	}
	if err := runner.AddHybrid("embedding", processor.NewEmbeddingProcessor(os.Getenv("EMBEDDING_URL")), qdrantSink); err != nil {
		log.Fatalf("Failed to add node 'embedding': %v", err)
	}

	if err := runner.Connect("start", "crawler"); err != nil {
		log.Fatalf("Graph wiring failed: %v", err)
	}
	if err := runner.Connect("crawler", "discovery"); err != nil {
		log.Fatalf("Graph wiring failed: %v", err)
	}
	if err := runner.Connect("crawler", "security"); err != nil {
		log.Fatalf("Graph wiring failed: %v", err)
	}

	if err := runner.Connect("security", "chunker"); err != nil {
		log.Fatalf("Graph wiring failed: %v", err)
	}
	if err := runner.Connect("chunker", "enrichment"); err != nil {
		log.Fatalf("Graph wiring failed: %v", err)
	}
	if err := runner.Connect("enrichment", "metadata"); err != nil {
		log.Fatalf("Graph wiring failed: %v", err)
	}
	if err := runner.Connect("metadata", "persist_pg"); err != nil {
		log.Fatalf("Graph wiring failed: %v", err)
	}
	if err := runner.Connect("metadata", "embedding"); err != nil {
		log.Fatalf("Graph wiring failed: %v", err)
	}

	log.Println("[Graph] Worker Topology constructed. Starting engine...")
	if err := runner.Run(ctx); err != nil {
		log.Printf("Worker stopped: %v", err)
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
	streamName := "CRAWL_JOBS"
	streamSubject := "crawl.>"

	_, err = nt.JS.StreamInfo(streamName)
	if err != nil {
		log.Printf("[NATS] Creating stream %s for subject %s...", streamName, streamSubject)
		_, err = nt.JS.AddStream(&nats.StreamConfig{
			Name:      streamName,
			Subjects:  []string{streamSubject},
			Retention: nats.WorkQueuePolicy,
		})
		if err != nil {
			pg.Close()
			rdb.Close()
			nt.Close()
			return nil, fmt.Errorf("failed to create nats stream: %w", err)
		}
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
