package source

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/oranjParker/Rarefactor/internal/core"
)

type NatsSource struct {
	JS      nats.JetStreamContext
	Subject string
	Queue   string
}

func NewNatsSource(js nats.JetStreamContext, subject, queue string) *NatsSource {
	return &NatsSource{
		JS:      js,
		Subject: subject,
		Queue:   queue,
	}
}

func (n *NatsSource) Stream(ctx context.Context) (<-chan *core.Document[string], error) {
	out := make(chan *core.Document[string])

	sub, err := n.JS.QueueSubscribeSync(n.Subject, n.Queue)
	if err != nil {
		return nil, fmt.Errorf("nats subscription failed: %w", err)
	}

	go func() {
		defer close(out)
		defer sub.Unsubscribe()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := sub.NextMsg(time.Second)
				if err != nil {
					continue
				}

				var doc core.Document[string]
				if err := json.Unmarshal(msg.Data, &doc); err != nil {
					// Fallback for raw strings (legacy or external producers)
					doc = core.Document[string]{
						ID:        string(msg.Data),
						Source:    "web",
						CreatedAt: time.Now(),
						Metadata:  make(map[string]any),
					}
				}

				if doc.Metadata == nil {
					doc.Metadata = make(map[string]any)
				}

				select {
				case out <- &doc:
					msg.Ack()
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}
