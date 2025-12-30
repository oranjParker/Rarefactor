package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"runtime"

	"github.com/oranjParker/Rarefactor/internal/database"
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

	lis, _ := net.Listen("tcp", ":50051")
	grpcServer := grpc.NewServer()

	// pb.RegisterRarefactorServiceServer(grpcServer, &MyServer{db: pool})

	go func() {
		log.Println("Starting gRPC server on :50051")
		grpcServer.Serve(lis)
	}()

	// 4. Start gRPC-Gateway (HTTP/1.1 Proxy)
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// err = pb.RegisterRarefactorServiceHandlerFromEndpoint(ctx, mux, "localhost:50051", opts)

	log.Println("Starting HTTP Gateway on :8000")
	http.ListenAndServe(":8000", mux)
}
