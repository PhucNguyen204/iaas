package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/pkg/env"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/pkg/logger"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type IKafkaProducer interface {
	PublishEvent(ctx context.Context, event InfrastructureEvent) error
	Close() error
}

type InfrastructureEvent struct {
	InstanceID string                 `json:"instance_id"`
	UserID     string                 `json:"user_id"`
	Type       string                 `json:"type"`
	Action     string                 `json:"action"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type kafkaProducer struct {
	writer *kafka.Writer
	logger logger.ILogger
}

func NewKafkaProducer(env env.KafkaEnv, logger logger.ILogger) IKafkaProducer {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(env.Brokers...),
		Topic:                  env.Topic,
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: true,
		Async:                  true,
		BatchSize:              100,
		BatchTimeout:           10 * time.Millisecond,
		RequiredAcks:           1,
		MaxAttempts:            3,
		WriteBackoffMin:        100 * time.Millisecond,
		WriteBackoffMax:        1 * time.Second,
		Compression:            kafka.Snappy,
	}

	return &kafkaProducer{
		writer: writer,
		logger: logger,
	}
}

func (kp *kafkaProducer) PublishEvent(ctx context.Context, event InfrastructureEvent) error {
	event.Timestamp = time.Now()

	eventBytes, err := json.Marshal(event)
	if err != nil {
		kp.logger.Error("failed to marshal event", zap.Error(err))
		return err
	}

	msg := kafka.Message{
		Key:   []byte(event.InstanceID),
		Value: eventBytes,
		Time:  event.Timestamp,
	}

	if err := kp.writer.WriteMessages(ctx, msg); err != nil {
		kp.logger.Error("failed to write message to kafka", zap.Error(err))
		return err
	}

	kp.logger.Info("event published to kafka",
		zap.String("instance_id", event.InstanceID),
		zap.String("action", event.Action))

	return nil
}

func (kp *kafkaProducer) Close() error {
	return kp.writer.Close()
}
