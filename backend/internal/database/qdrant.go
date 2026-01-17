package database

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

type QdrantClient struct {
	Client *qdrant.Client
}

func NewQdrantClient(ctx context.Context) (*QdrantClient, error) {
	addr := os.Getenv("QDRANT_URL")
	if addr == "" {
		fmt.Println("[Qdrant] WARNING: QDRANT_URL env is empty, defaulting to localhost:6334")
		addr = "localhost:6334"
	}

	fmt.Printf("[Qdrant] Attempting to connect to: %s\n", addr)

	host := "localhost"
	port := 6334

	if strings.Contains(addr, ":") {
		h, pStr, err := net.SplitHostPort(addr)
		if err == nil {
			host = h
			if p, err := strconv.Atoi(pStr); err == nil {
				port = p
			}
		} else {
			host = addr
		}
	} else {
		host = addr
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init qdrant client: %w", err)
	}

	return &QdrantClient{Client: client}, nil
}

func (q *QdrantClient) EnsureCollection(ctx context.Context, name string) error {
	collections, err := q.Client.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	exists := false
	for _, c := range collections {
		if c == name {
			exists = true
			break
		}
	}

	if exists {
		fmt.Printf("[Qdrant] Collection '%s' verified\n", name)
		return nil
	}

	fmt.Printf("[Qdrant] Creating collection '%s' (768 dims)\n", name)
	return q.Client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: name,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     768,
			Distance: qdrant.Distance_Cosine,
		}),
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
