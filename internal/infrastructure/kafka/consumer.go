package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"livechat-ws/internal/domain"

	"github.com/segmentio/kafka-go"
)

type MessageHandler interface {
	HandleNewMessage(msg domain.ChatMessage)
	HandleTypingIndicator(msg domain.TypingMessage)
	HandleConnectionStatus(msg domain.ConnectionStatusMessage)
}

type KafkaConsumer struct {
	readers []*kafka.Reader
	handler MessageHandler
}

func NewKafkaConsumer(brokers []string, groupID string, topics []string, handler MessageHandler) *KafkaConsumer {
	var readers []*kafka.Reader

	for _, topic := range topics {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       1,                      // Read immediately, don't wait for batches
			MaxBytes:       10e6,                   // 10MB max
			CommitInterval: 100 * time.Millisecond, // Commit every 100ms instead of 1s
			StartOffset:    kafka.LastOffset,
			MaxWait:        100 * time.Millisecond, // Max wait 100ms for new data
		})
		readers = append(readers, reader)
	}

	return &KafkaConsumer{
		readers: readers,
		handler: handler,
	}
}

func (k *KafkaConsumer) Start(ctx context.Context) error {
	// Start consumers for each topic in separate goroutines
	for i := range k.readers {
		go func(readerIndex int) {
			// Recovery dari panic untuk mencegah crash goroutine
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic in Kafka consumer goroutine %d: %v", readerIndex, r)
				}
			}()

			reader := k.readers[readerIndex]
			defer reader.Close()

			for {
				select {
				case <-ctx.Done():
					log.Printf("Kafka consumer for topic stopping...")
					return
				default:
					m, err := reader.ReadMessage(ctx)
					if err != nil {
						// Handle specific Kafka errors more gracefully
						if err.Error() == "[27] Rebalance In Progress: the coordinator has begun rebalancing the group, the client should rejoin the group" {
							log.Printf("Kafka rebalance in progress, continuing...")
							continue
						}
						if err.Error() == "[5] Leader Not Available: the cluster is in the middle of a leadership election and there is currently no leader for this partition and hence it is unavailable for writes" {
							log.Printf("Kafka leader election in progress, continuing...")
							continue
						}
						log.Printf("Error reading Kafka message: %v", err)
						continue
					}

					if k.handler != nil {
						k.handleMessage(m.Topic, m.Value)
					}
				}
			}
		}(i)
	}

	return nil
}

func (k *KafkaConsumer) handleMessage(topic string, value []byte) {
	// Recovery dari panic untuk mencegah crash consumer
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in handleMessage for topic %s: %v", topic, r)
		}
	}()

	log.Printf("Received Kafka message from topic %s", topic)

	switch topic {
	case "chat-messages":
		log.Printf("Processing chat message from Kafka: %s", string(value))
		var chatMsg domain.ChatMessage
		if err := json.Unmarshal(value, &chatMsg); err != nil {
			log.Printf("Error unmarshaling chat message: %v", err)
			log.Printf("Raw message: %s", string(value))
			return
		}
		log.Printf("Successfully unmarshaled chat message: ID=%s, SessionID=%s, SenderType=%s",
			chatMsg.ID, chatMsg.SessionID, chatMsg.SenderType)
		k.handler.HandleNewMessage(chatMsg)

	case "typing-indicators":
		var typingMsg domain.TypingMessage
		if err := json.Unmarshal(value, &typingMsg); err != nil {
			log.Printf("Error unmarshaling typing message: %v", err)
			return
		}
		k.handler.HandleTypingIndicator(typingMsg)

	case "connection-status":
		var statusMsg domain.ConnectionStatusMessage
		if err := json.Unmarshal(value, &statusMsg); err != nil {
			log.Printf("Error unmarshaling connection status message: %v", err)
			return
		}
		k.handler.HandleConnectionStatus(statusMsg)

	default:
		log.Printf("Unknown topic: %s", topic)
	}
}

func (k *KafkaConsumer) Close() error {
	for i := range k.readers {
		if err := k.readers[i].Close(); err != nil {
			log.Printf("Error closing Kafka reader: %v", err)
		}
	}
	return nil
}
