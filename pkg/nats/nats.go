// Package nats provides a thin wrapper around NATS JetStream for publishing
// and consuming events across services.
package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Client wraps a NATS connection and JetStream context.
type Client struct {
	conn *nats.Conn
	js   jetstream.JetStream
}

// Connect establishes a NATS connection and returns a Client.
func Connect(url string) (*Client, error) {
	nc, err := nats.Connect(url,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("nats.Connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("nats.Connect: jetstream: %w", err)
	}

	return &Client{conn: nc, js: js}, nil
}

// Close drains and closes the NATS connection.
func (c *Client) Close() {
	_ = c.conn.Drain()
}

// Publisher publishes JSON-encoded events to NATS JetStream subjects.
type Publisher struct {
	js jetstream.JetStream
}

// NewPublisher creates a Publisher from a Client.
func NewPublisher(c *Client) *Publisher {
	return &Publisher{js: c.js}
}

// Publish serializes payload as JSON and publishes to subject.
func (p *Publisher) Publish(ctx context.Context, subject string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("Publisher.Publish: marshal: %w", err)
	}

	_, err = p.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("Publisher.Publish %s: %w", subject, err)
	}
	return nil
}

// ConsumerConfig holds configuration for creating a durable pull consumer.
type ConsumerConfig struct {
	Stream       string
	ConsumerName string
	Subjects     []string
	MaxDeliver   int
}

// CreateOrUpdateStream ensures a JetStream stream exists with the given subjects.
func (c *Client) CreateOrUpdateStream(ctx context.Context, name string, subjects []string) error {
	_, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     name,
		Subjects: subjects,
		Storage:  jetstream.FileStorage,
		MaxAge:   7 * 24 * time.Hour,
	})
	if err != nil {
		return fmt.Errorf("nats.CreateOrUpdateStream %s: %w", name, err)
	}
	return nil
}

// CreateOrUpdateConsumer creates a durable pull consumer on a stream.
func (c *Client) CreateOrUpdateConsumer(ctx context.Context, cfg ConsumerConfig) (jetstream.Consumer, error) {
	maxDeliver := cfg.MaxDeliver
	if maxDeliver == 0 {
		maxDeliver = 5
	}

	consumer, err := c.js.CreateOrUpdateConsumer(ctx, cfg.Stream, jetstream.ConsumerConfig{
		Durable:       cfg.ConsumerName,
		FilterSubjects: cfg.Subjects,
		MaxDeliver:    maxDeliver,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("nats.CreateOrUpdateConsumer %s: %w", cfg.ConsumerName, err)
	}
	return consumer, nil
}

// HandleFunc is the signature for event handler callbacks.
type HandleFunc func(ctx context.Context, subject string, data []byte) error

// StartConsumer runs a pull-based consumer loop until ctx is done.
func StartConsumer(ctx context.Context, consumer jetstream.Consumer, handler HandleFunc) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msgs, err := consumer.Fetch(10, jetstream.FetchMaxWait(2*time.Second))
		if err != nil {
			if err == nats.ErrTimeout {
				continue
			}
			return fmt.Errorf("nats.StartConsumer fetch: %w", err)
		}

		for msg := range msgs.Messages() {
			if err := handler(ctx, msg.Subject(), msg.Data()); err != nil {
				_ = msg.Nak()
				continue
			}
			_ = msg.Ack()
		}
		if msgs.Error() != nil && msgs.Error() != nats.ErrTimeout {
			return fmt.Errorf("nats.StartConsumer: %w", msgs.Error())
		}
	}
}
