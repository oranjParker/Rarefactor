package source

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/oranjParker/Rarefactor/internal/core"
)

type NatsSource struct {
	JS      jetstream.JetStream
	Subject string
	Queue   string
}

func NewNatsSource(js jetstream.JetStream, subject, queue string) *NatsSource {
	return &NatsSource{
		JS:      js,
		Subject: subject,
		Queue:   queue,
	}
}

func (n *NatsSource) Stream(ctx context.Context) (<-chan *core.Document[string], error) {
	out := make(chan *core.Document[string])

	consumer, err := n.JS.CreateOrUpdateConsumer(ctx, "CRAWL_JOBS", jetstream.ConsumerConfig{
		Durable:       n.Queue,
		FilterSubject: n.Subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("nats consumer setup failed: %w", err)
	}

	iter, err := consumer.Messages()
	if err != nil {
		return nil, fmt.Errorf("nats consumer iterator failed: %w", err)
	}

	go func() {
		defer close(out)
		defer iter.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := iter.Next()
				if err != nil {
					log.Printf("[NATS Source] NextMsg error: %v", err)
					continue
				}

				var doc core.Document[string]
				msgData := msg.Data()
				if err := json.Unmarshal(msgData, &doc); err != nil {
					log.Printf("[NATS Source] Malformed JSON, Terminating msg: %v", err)
					msg.Term()
					continue
				}

				if doc.Metadata == nil {
					doc.Metadata = make(map[string]any)
				}

				var once sync.Once
				ack := func() {
					once.Do(func() {
						if err := msg.Ack(); err != nil {
							log.Printf("[NATS Source] Failed to Ack msg for %s: %v", doc.ID, err)
						}
					})
				}

				nack := func() {
					once.Do(func() {
						if err := msg.Nak(); err != nil {
							log.Printf("[NATS Source] Failed to Nack msg for %s: %v", doc.ID, err)
						}
					})
				}

				doc.CT = core.NewCompletionTracker(ack, nack)

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
