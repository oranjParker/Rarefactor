package source

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
)

type LocalSource struct {
	RootPath   string
	Extensions []string
}

func NewLocalSource(root string, exts ...string) *LocalSource {
	return &LocalSource{
		RootPath:   root,
		Extensions: exts,
	}
}

func (l *LocalSource) Stream(ctx context.Context) (<-chan string, error) {
	out := make(chan string)

	go func() {
		defer close(out)

		err := filepath.WalkDir(l.RootPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			if len(l.Extensions) > 0 {
				ext := strings.ToLower(filepath.Ext(path))
				match := false
				for _, validExt := range l.Extensions {
					if ext == validExt {
						match = true
						break
					}
				}
				if !match {
					return nil
				}
			}

			select {
			case out <- path:
			case <-ctx.Done():
				return filepath.SkipAll
			}
			return nil
		})

		if err != nil {
			// In a real source, we might log this or send to an error channel,
			// but Stream signature only allows returning error on startup.
			// Runtime errors during walking are suppressed here for simplicity.
		}
	}()

	return out, nil
}
