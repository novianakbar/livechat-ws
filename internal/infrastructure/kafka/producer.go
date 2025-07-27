package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"livechat-ws/internal/domain"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	Writer *kafka.Writer
}

func NewKafkaProducer(broker, defaultTopic string) *KafkaProducer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(broker),
		Balancer: &kafka.LeastBytes{},
		// Optimize for low latency
		BatchSize:    1,                    // Send immediately, don't batch
		BatchTimeout: 0 * time.Millisecond, // 1ms timeout
		RequiredAcks: 1,                    // Wait for leader acknowledgment only
		Async:        false,                // Synchronous for immediate sending
	}
	return &KafkaProducer{Writer: writer}
}

func (k *KafkaProducer) SendMessage(ctx context.Context, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Determine topic based on message type
	topic := k.getTopicForMessage(message)

	msg := kafka.Message{
		Topic: topic,
		Value: data,
	}

	err = k.Writer.WriteMessages(ctx, msg)
	if err != nil {
		log.Printf("Failed to send message to Kafka topic %s: %v", topic, err)
		return err
	}

	log.Printf("Message sent to Kafka topic %s successfully", topic)
	return nil
}

func (k *KafkaProducer) getTopicForMessage(message interface{}) string {
	switch message.(type) {
	case domain.ChatMessage:
		return "chat-messages"
	case domain.TypingMessage:
		return "typing-indicators"
	case domain.ConnectionStatusMessage:
		return "connection-status"
	default:
		return "chat-messages" // fallback to default topic
	}
}

func (k *KafkaProducer) Close() error {
	return k.Writer.Close()
}
