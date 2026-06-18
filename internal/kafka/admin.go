package kafka

import (
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

func EnsureTopics(ctx context.Context, brokers []string, topics ...string) error {
	if len(brokers) == 0 || len(topics) == 0 {
		return nil
	}

	dialer := &kafkago.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", brokers[0])
	if err != nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}

	controllerAddr := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
	controllerConn, err := dialer.DialContext(ctx, "tcp", controllerAddr)
	if err != nil {
		return err
	}
	defer controllerConn.Close()

	cfgs := make([]kafkago.TopicConfig, 0, len(topics))
	for _, topic := range topics {
		cfgs = append(cfgs, kafkago.TopicConfig{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
	}

	if err := controllerConn.CreateTopics(cfgs...); err != nil {
		if errors.Is(err, kafkago.TopicAlreadyExists) {
			return nil
		}
		return err
	}
	return nil
}
