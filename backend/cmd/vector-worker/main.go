package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/oranjParker/Rarefactor/internal/core"
	"github.com/oranjParker/Rarefactor/internal/database"
	"github.com/oranjParker/Rarefactor/internal/processor"
	"github.com/oranjParker/Rarefactor/internal/sink"
	"github.com/oranjParker/Rarefactor/internal/source"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	nt, _ := database.NewNatsConnection()
	pg, _ := database.NewPool(ctx)
	qdb, _ := database.NewQdrantClient(ctx)
	defer nt.Close()
	defer pg.Close()
	defer qdb.Close()

	natsSrc := source.NewNatsSource(nt.JS, "vector.jobs", "vector-group")

	qdrantSink := sink.NewQdrantSink(qdb, "documents")

	lookupProc := &core.FunctionalProcessor[*core.Document[string], *core.Document[string]]{
		Fn: func(ctx context.Context, doc *core.Document[string]) ([]*core.Document[string], error) {
			var title, content string
			var metadata map[string]any
			err := pg.QueryRow(ctx, "SELECT title, content, metadata FROM documents WHERE url = $1", doc.ID).
				Scan(&title, &content, &metadata)

			if err != nil {
				return nil, err
			}

			doc.Content = content
			if doc.Metadata == nil {
				doc.Metadata = make(map[string]any)
			}

			for k, v := range metadata {
				doc.Metadata[k] = v
			}

			doc.Metadata["title"] = title
			return []*core.Document[string]{doc}, err
		},
	}

	embedder := processor.NewEmbeddingProcessor(os.Getenv("EMBEDDING_URL"))

	runner := core.NewPipelineRunner(natsSrc, qdrantSink, core.PipelineConfig{
		Concurrency: 5,
		Name:        "Vector-Ingestion-Pipeline",
	})

	runner.AddProcessor(lookupProc)
	runner.AddProcessor(embedder)

	log.Println("Vector Worker active. Processing embedding stream...")
	if err := runner.Run(ctx); err != nil {
		log.Printf("Vector Pipeline exited: %v", err)
	}
}
