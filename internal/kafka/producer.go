package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafkago.Writer
}

func NewProducer(brokers []string) *Producer {
	return &Producer{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(brokers...),
			Balancer:     &kafkago.LeastBytes{},
			RequiredAcks: kafkago.RequireOne,
			Async:        false,
		},
	}
}

func (p *Producer) PublishJSON(ctx context.Context, topic, key string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := kafkago.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: body,
		Time:  time.Now().UTC(),
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if err := p.writer.WriteMessages(ctx, msg); err == nil {
			return nil
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt) * 150 * time.Millisecond):
		}
	}
	return fmt.Errorf("publish %s: %w", topic, lastErr)
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
