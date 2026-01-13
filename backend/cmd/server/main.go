package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pb "github.com/oranjParker/Rarefactor/generated/protos/v1"
	"github.com/oranjParker/Rarefactor/internal/database"
	"github.com/oranjParker/Rarefactor/internal/search"
	"github.com/oranjParker/Rarefactor/internal/server"
	"github.com/oranjParker/Rarefactor/internal/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}
	defer pool.Close()

	rdb, err := database.NewRedisClient(ctx)
	if err != nil {
		log.Fatalf("Redis failed to initialize: %v", err)
	}
	defer rdb.Close()

	qdb, err := database.NewQdrantClient(ctx)
	if err != nil {
		log.Fatalf("Qdrant failed: %v", err)
	}
	defer qdb.Close()

	if err := qdb.EnsureCollection(ctx, "documents"); err != nil {
		log.Printf("Warning: Qdrant collection setup: %v", err)
	}

	embedder := search.NewEmbedder()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	searchSrv := server.NewSearchServer(pool, rdb, qdb, embedder)

	pb.RegisterSearchEngineServiceServer(grpcServer, searchSrv)

	crawlerSrv := server.NewCrawlerServer(pool, rdb, qdb, embedder)
	pb.RegisterCrawlerServiceServer(grpcServer, crawlerSrv)

	go func() {
		log.Println("Starting gRPC server on :50051")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	// 4. Start gRPC-Gateway (HTTP/1.1 Proxy)
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	err = pb.RegisterCrawlerServiceHandlerFromEndpoint(ctx, mux, "localhost:50051", opts)
	if err != nil {
		log.Fatalf("Failed to register Search Gateway: %v", err)
	}
	err = pb.RegisterSearchEngineServiceHandlerFromEndpoint(ctx, mux, "localhost:50051", opts)
	if err != nil {
		log.Fatalf("Failed to register Search Gateway: %v", err)
	}

	httpServer := &http.Server{
		Addr:    ":8000",
		Handler: utils.AllowCORS(mux),
	}
	log.Println("Starting HTTP Gateway on :8000")
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("failed to serve Gateway: %v", err)
	}
}
