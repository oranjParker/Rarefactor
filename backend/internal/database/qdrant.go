package database

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

type QdrantClient struct {
	Client *qdrant.Client
}

func NewQdrantClient(ctx context.Context) (*QdrantClient, error) {
	addr := os.Getenv("QDRANT_GRPC_URL")
	if addr == "" {
		addr = "localhost:6334"
	}

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
		portStr = "6334"
	}

	port, _ := strconv.Atoi(portStr)

	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize qdrant client: %w", err)
	}

	return &QdrantClient{
		Client: client,
	}, nil
}

func (q *QdrantClient) EnsureCollection(ctx context.Context, name string) error {
	exists, err := q.Client.CollectionExists(ctx, name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return q.Client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: name,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     768,
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	})
}

func (q *QdrantClient) Upsert(ctx context.Context, collection, url, title, snippet string, vector []float32) error {
	id := uuid.NewMD5(uuid.NameSpaceURL, []byte(url)).String()

	_, err := q.Client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collection,
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewIDUUID(id),
				Vectors: qdrant.NewVectors(vector...),
				Payload: qdrant.NewValueMap(map[string]any{
					"url":     url,
					"title":   title,
					"snippet": snippet,
				}),
			},
		},
	})

	return err
}

func (q *QdrantClient) Query(ctx context.Context, collection string, vector []float32, limit uint64) ([]*qdrant.ScoredPoint, error) {
	res, err := q.Client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: collection,
		Query:          qdrant.NewQuery(vector...),
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (q *QdrantClient) Close() error {
	return q.Client.Close()
}
