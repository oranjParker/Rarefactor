package sink

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/oranjParker/Rarefactor/internal/core"
)

type NatsSink struct {
	JS      nats.JetStreamContext
	Subject string
}

func NewNatsSink(js nats.JetStreamContext, subject string) *NatsSink {
	return &NatsSink{
		JS:      js,
		Subject: subject,
	}
}

func (n *NatsSink) Write(ctx context.Context, doc *core.Document[string]) error {
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("nats marshal failed: %w", err)
	}

	_, err = n.JS.Publish(n.Subject, data)
	return err
}

func (n *NatsSink) Close() error {
	return nil
}
