package server

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/oranjParker/Rarefactor/generated/protos/v1"
	"github.com/oranjParker/Rarefactor/internal/database"
	"github.com/oranjParker/Rarefactor/internal/search"
	"github.com/redis/go-redis/v9"
)

const (
	AutocompleteKey       = "rarefactor:autocomplete"
	GlobalSearchScoresKey = "global_search_scores"
	CollectionName        = "documents"
)

type SearchServer struct {
	dbPool   *pgxpool.Pool
	rdb      *redis.Client
	qdb      *database.QdrantClient
	embedder *search.Embedder
}

type scoredTerm struct {
	term  string
	score float64
}

func NewSearchServer(db *pgxpool.Pool, rdb *redis.Client, qdb *database.QdrantClient, emb *search.Embedder) *SearchServer {
	return &SearchServer{
		dbPool:   db,
		rdb:      rdb,
		qdb:      qdb,
		embedder: emb,
	}
}

func (s *SearchServer) Autocomplete(ctx context.Context, req *pb.AutocompleteRequest) (*pb.AutocompleteResponse, error) {
	start := time.Now()
	prefix := strings.ToLower(strings.TrimSpace(req.Prefix))
	if prefix == "" {
		return &pb.AutocompleteResponse{}, nil
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	terms, err := s.rdb.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:   AutocompleteKey,
		ByLex: true,
		Start: "[" + prefix,
		Stop:  "[" + prefix + "\xff",
		Count: int64(limit * 2),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("redis lookup failed: %w", err)
	}

	if len(terms) == 0 {
		return &pb.AutocompleteResponse{
			Suggestions: []string{},
			DurationMs:  float64(time.Since(start).Milliseconds()),
		}, nil
	}

	pipe := s.rdb.Pipeline()
	scoreCmds := make([]*redis.FloatCmd, len(terms))
	for i, term := range terms {
		scoreCmds[i] = pipe.ZScore(ctx, GlobalSearchScoresKey, term)
	}

	_, err = pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("ranking pipeline failed: %w", err)
	}

	scored := make([]scoredTerm, len(terms))
	for i, term := range terms {
		score, _ := scoreCmds[i].Result()
		scored[i] = scoredTerm{term: term, score: score}
	}

	slices.SortFunc(scored, func(a, b scoredTerm) int {
		if n := cmp.Compare(b.score, a.score); n != 0 {
			return n
		}
		return cmp.Compare(a.term, b.term)
	})

	suggestions := make([]string, 0, limit)
	for i := 0; i < len(scored) && i < int(limit); i++ {
		suggestions = append(suggestions, scored[i].term)
	}

	return &pb.AutocompleteResponse{
		Suggestions: suggestions,
		DurationMs:  float64(time.Since(start).Milliseconds()),
	}, nil
}

func (s *SearchServer) Search(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	query := strings.ToLower(strings.TrimSpace(req.Query))
	if query == "" {
		return &pb.SearchResponse{}, nil
	}

	_ = s.rdb.ZIncrBy(ctx, GlobalSearchScoresKey, 1.0, query).Err()

	vector, err := s.embedder.ComputeEmbeddings(ctx, query, true)
	if err != nil {
		return nil, fmt.Errorf("embedding failed: %w", err)
	}

	limit := uint64(req.Limit)
	if limit <= 0 {
		limit = 10
	}
	points, err := s.qdb.Query(ctx, CollectionName, vector, limit)
	if err != nil {
		return nil, fmt.Errorf("qdrant query failed: %w", err)
	}

	results := make([]*pb.Document, 0, len(points))
	for _, p := range points {
		payload := p.GetPayload()
		results = append(results, &pb.Document{
			Url:   payload["url"].GetStringValue(),
			Title: payload["title"].GetStringValue(),
			Score: p.GetScore(),
		})
	}
	return &pb.SearchResponse{
		Results:   results,
		TotalHits: int32(len(results)),
	}, nil
}

func (s *SearchServer) UpdateDocument(ctx context.Context, req *pb.UpdateDocumentRequest) (*pb.Document, error) {
	// s.rdb.ZAdd(ctx, AutocompleteKey, redis.Z{Member: req.Document.Title, Score: 0})

	return req.Document, nil
}
