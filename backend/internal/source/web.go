package source

import "context"

type WebSource struct {
	SeedURL string
}

func NewWebSource(seedURL string) *WebSource {
	return &WebSource{SeedURL: seedURL}
}

func (w *WebSource) Stream(ctx context.Context) (<-chan string, error) {
	out := make(chan string)

	go func() {
		defer close(out)

		select {
		case out <- w.SeedURL:
		case <-ctx.Done():
			return
		}
	}()

	return out, nil
}
