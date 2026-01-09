package main

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pb "github.com/oranjParker/Rarefactor/generated/protos/v1"
	"github.com/oranjParker/Rarefactor/internal/database"
	"github.com/oranjParker/Rarefactor/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx := context.Background()

	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer pool.Close()

	rdb, err := database.NewRedisClient(ctx)
	if err != nil {
		log.Fatalf("Redis failed to initialize: %v", err)
	}
	defer rdb.Close()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// pb.RegisterSearchEngineServiceServer(grpcServer, &SearchServer{db : pool})
	crawlerSrv := server.NewCrawlerServer(pool, rdb)
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

	log.Println("Starting HTTP Gateway on :8000")
	if err := http.ListenAndServe("0.0.0.0:8000", mux); err != nil {
		log.Fatalf("failed to serve Gateway: %v", err)
	}
}
