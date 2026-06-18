package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

type MessageHandler func(context.Context, kafkago.Message) error

type Consumer struct {
	reader   *kafkago.Reader
	producer *Producer
	dlqTopic string
	logger   *slog.Logger
}

func NewConsumer(brokers []string, groupID, topic, dlqTopic string, producer *Producer, logger *slog.Logger) *Consumer {
	return &Consumer{
		reader: kafkago.NewReader(kafkago.ReaderConfig{
			Brokers:        brokers,
			GroupID:        groupID,
			Topic:          topic,
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: 0,
			StartOffset:    kafkago.FirstOffset,
		}),
		producer: producer,
		dlqTopic: dlqTopic,
		logger:   logger,
	}
}

func (c *Consumer) Run(ctx context.Context, handler MessageHandler) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			c.logger.Error("kafka fetch failed", "error", err)
			continue
		}

		if err := c.handleWithRetry(ctx, msg, handler); err != nil {
			c.logger.Error("message moved to dlq", "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset, "error", err)
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error("kafka commit failed", "error", err)
		}
	}
}

func (c *Consumer) handleWithRetry(ctx context.Context, msg kafkago.Message, handler MessageHandler) error {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if err := handler(ctx, msg); err == nil {
			return nil
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt) * 250 * time.Millisecond):
		}
	}

	if c.producer != nil && c.dlqTopic != "" {
		dlq := map[string]any{
			"source_topic": msg.Topic,
			"source_key":   string(msg.Key),
			"payload":      json.RawMessage(msg.Value),
			"error":        lastErr.Error(),
			"failed_at":    time.Now().UTC(),
		}
		if err := c.producer.PublishJSON(ctx, c.dlqTopic, string(msg.Key), dlq); err != nil {
			return fmt.Errorf("handler failed: %w; dlq publish failed: %v", lastErr, err)
		}
	}
	return lastErr
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
