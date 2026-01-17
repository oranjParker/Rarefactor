package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/oranjParker/Rarefactor/generated/protos/v1"
	"github.com/oranjParker/Rarefactor/internal/crawler"
	"github.com/oranjParker/Rarefactor/internal/database"
	"github.com/oranjParker/Rarefactor/internal/search"
	"github.com/oranjParker/Rarefactor/internal/utils"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}

type AppDependencies struct {
	Pool     *pgxpool.Pool
	Redis    *redis.Client
	Qdrant   *database.QdrantClient
	Embedder *search.Embedder
}

func run(ctx context.Context) error {
	deps, err := setupDependencies(ctx)
	if err != nil {
		return err
	}
	if deps.Pool != nil {
		defer deps.Pool.Close()
	}
	if deps.Redis != nil {
		defer deps.Redis.Close()
	}
	if deps.Qdrant != nil {
		defer deps.Qdrant.Close()
	}

	return runWithDeps(ctx, deps)
}

func setupDependencies(ctx context.Context) (*AppDependencies, error) {
	initCtx, initCancel := context.WithTimeout(ctx, 30*time.Second)
	defer initCancel()

	pool, err := database.NewPool(initCtx)
	if err != nil {
		return nil, fmt.Errorf("postgres init: %w", err)
	}

	rdb, err := database.NewRedisClient(initCtx)
	if err != nil {
		if pool != nil {
			pool.Close()
		}
		return nil, fmt.Errorf("redis init: %w", err)
	}

	qdb, err := database.NewQdrantClient(initCtx)
	if err != nil {
		if pool != nil {
			pool.Close()
		}
		if rdb != nil {
			rdb.Close()
		}
		return nil, fmt.Errorf("qdrant init: %w", err)
	}

	if err := qdb.EnsureCollection(initCtx, "documents"); err != nil {
		log.Printf("Warning: Qdrant collection setup: %v", err)
	}

	embedder := search.NewEmbedder()

	return &AppDependencies{
		Pool:     pool,
		Redis:    rdb,
		Qdrant:   qdb,
		Embedder: embedder,
	}, nil
}

func runWithDeps(ctx context.Context, deps *AppDependencies) error {
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8000"
	}

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		return fmt.Errorf("failed to listen on :%s: %w", grpcPort, err)
	}

	grpcServer := grpc.NewServer()

	searchSrv := search.NewSearchServer(deps.Pool, deps.Redis, deps.Qdrant, deps.Embedder)
	pb.RegisterSearchEngineServiceServer(grpcServer, searchSrv)

	crawlerSrv := crawler.NewCrawlerServer(ctx, deps.Pool, deps.Redis, deps.Qdrant, deps.Embedder)
	pb.RegisterCrawlerServiceServer(grpcServer, crawlerSrv)

	reflection.Register(grpcServer)

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	if err := pb.RegisterCrawlerServiceHandlerFromEndpoint(ctx, mux, "localhost:"+grpcPort, opts); err != nil {
		return fmt.Errorf("failed to register Crawler Gateway: %w", err)
	}
	if err := pb.RegisterSearchEngineServiceHandlerFromEndpoint(ctx, mux, "localhost:"+grpcPort, opts); err != nil {
		return fmt.Errorf("failed to register Search Gateway: %w", err)
	}

	httpServer := &http.Server{
		Addr:    ":" + httpPort,
		Handler: utils.AllowCORS(mux),
	}

	errChan := make(chan error, 2)

	go func() {
		log.Printf("Starting gRPC server on :%s\n", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			errChan <- fmt.Errorf("grpc server error: %w", err)
		}
	}()

	go func() {
		log.Printf("Starting HTTP Gateway on :%s\n", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("http gateway error: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("Shutting down servers...")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		grpcServer.GracefulStop()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http shutdown error: %w", err)
		}
		return nil

	case err := <-errChan:
		return err
	}
}
