package database

import (
	"fmt"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

type NatsConn struct {
	Conn *nats.Conn
	JS   nats.JetStreamContext
}

func NewNatsConnection() (*NatsConn, error) {
	url := os.Getenv("NATS_URL")
	if url == "" {
		url = nats.DefaultURL
	}

	nc, err := nats.Connect(url,
		nats.Timeout(10*time.Second),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS at %s: %w", url, err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to initialize JetStream: %w", err)
	}

	return &NatsConn{
		Conn: nc,
		JS:   js,
	}, nil
}

func (n *NatsConn) Close() {
	if n.Conn != nil {
		n.Conn.Close()
	}
}
