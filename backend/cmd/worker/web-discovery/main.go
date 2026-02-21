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

	// =========================================================================
	// CONTROL PLANE: Start gRPC Server (Simulated API Pod)
	// =========================================================================
	go func() {
		listener, err := net.Listen("tcp", GRPC_PORT)
		if err != nil {
			log.Fatalf("[Control Plane] Failed to listen on %s: %v", GRPC_PORT, err)
		}

		grpcServer := grpc.NewServer()
		crawlerService := crawler.NewCrawlerService(deps.Postgres, deps.Nats.JS)
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
	discoverySrc := source.NewNatsSource(deps.Nats.JS, "crawl.jobs", "discovery-group")
	pgSink := sink.NewPostgresSink(deps.Postgres, 50, 5*time.Second)
	defer pgSink.Close()

	discoverySink := sink.NewNatsSink(deps.Nats.JS, "crawl.jobs")
	enrichmentSink := sink.NewNatsSink(deps.Nats.JS, "crawl.enrichment")

	runner := core.NewGraphRunner("Rarefactor-V2", discoverySrc, 5)

	if err := runner.AddProcessor("start", processor.NewPolitenessProcessor(deps.Redis, "RarefactorBot/2.0", 3, 1000, false)); err != nil {
		log.Fatalf("Failed to add node 'start': %v", err)
	}
	if err := runner.AddProcessor("crawler", processor.NewSmartCrawlerProcessor()); err != nil {
		log.Fatalf("Failed to add node 'crawler': %v", err)
	}
	if err := runner.AddHybrid("discovery", processor.NewDiscoveryProcessor(), discoverySink); err != nil {
		log.Fatalf("Failed to add node 'discovery': %v", err)
	}
	if err := runner.AddProcessor("security", processor.NewSecurityProcessor(false)); err != nil { // false = don't fail, just flag
		log.Fatalf("Failed to add node 'security': %v", err)
	}
	if err := runner.AddHybrid("chunker", processor.NewChunkerProcessor(4000, 400), pgSink); err != nil {
		log.Fatalf("Failed to add node 'chunker': %v", err)
	}

	if err := runner.AddHybrid("async_enrichment", processor.NewEnrichmentProcessor(), enrichmentSink); err != nil {
		log.Fatalf("Failed to add node 'async_enrichment': %v", err)
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
	if err := runner.Connect("chunker", "async_enrichment"); err != nil {
		log.Fatalf("Graph wiring failed: %v", err)
	}

	log.Println("[Graph] Worker Topology constructed. Starting engine...")
	if err := runner.Run(ctx); err != nil {
		log.Printf("[Graph] Worker stopped: %v", err)
	}
}

type WorkerDependencies struct {
	Nats     *database.NatsConn
	Redis    *redis.Client
	Postgres *pgxpool.Pool
}

func setupWorkerDependencies(ctx context.Context) (*WorkerDependencies, error) {
	deadline := time.Now().Add(120 * time.Second)

	var (
		pg  *pgxpool.Pool
		rdb *redis.Client
		nt  *database.NatsConn
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

		if rdb == nil {
			rdb, err = database.NewRedisClient(ctx)
			if err != nil {
				log.Printf("Redis not ready: %v", err)
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

		return &WorkerDependencies{
			Nats:     nt,
			Redis:    rdb,
			Postgres: pg,
		}, nil

	retry:
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("infrastructure initialization timed out after 120s")
}
