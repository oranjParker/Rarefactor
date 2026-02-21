package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
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

	sub, err := n.JS.QueueSubscribeSync(n.Subject, n.Queue, nats.AckExplicit())
	if err != nil {
		return nil, fmt.Errorf("nats subscription failed: %w", err)
	}

	if err := sub.SetPendingLimits(100000, 512*1024*1024); err != nil {
		log.Printf("[NATS Source] Warning: Could not set pending limits: %v", err)
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
					if errors.Is(err, nats.ErrTimeout) {
						continue
					}
					log.Printf("[NATS Source] NextMsg error: %v", err)
					continue
				}

				var doc core.Document[string]
				if err := json.Unmarshal(msg.Data, &doc); err != nil {
					log.Printf("[NATS Source] Malformed JSON, Terminating msg: %v", err)
					msg.Term()
					continue
				}

				if doc.Metadata == nil {
					doc.Metadata = make(map[string]any)
				}

				var once sync.Once
				doc.Ack = func() {
					once.Do(func() {
						if err := msg.Ack(); err != nil {
							log.Printf("[NATS Source] Failed to Ack msg for %s: %v", doc.ID, err)
						}
					})
				}

				select {
				case out <- &doc:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}
