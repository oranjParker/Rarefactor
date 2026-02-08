package processor

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/oranjParker/Rarefactor/internal/core"
)

type FileLoaderProcessor struct{}

func NewFileLoaderProcessor() *FileLoaderProcessor {
	return &FileLoaderProcessor{}
}

func (p *FileLoaderProcessor) Process(ctx context.Context, path string) ([]*core.Document[string], error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	doc := &core.Document[string]{
		ID:        path,
		Source:    "local_fs",
		Content:   string(content),
		CreatedAt: time.Now(),
		Metadata: map[string]any{
			"filename": path,
		},
	}

	return []*core.Document[string]{doc}, nil
}
