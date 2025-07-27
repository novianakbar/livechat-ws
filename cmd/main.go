package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"livechat-ws/internal/config"
	"livechat-ws/internal/delivery"
	"livechat-ws/internal/infrastructure/kafka"
	"livechat-ws/internal/infrastructure/redis"

	"github.com/joho/godotenv"
)

func main() {
	// Recovery global untuk mencegah crash aplikasi
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Application recovered from panic: %v", r)
			os.Exit(1)
		}
	}()

	_ = godotenv.Load()

	// Load configuration
	cfg := config.LoadConfig()

	log.Printf("Starting LiveChat WebSocket Server")
	log.Printf("Environment: %s", cfg.Environment)
	log.Printf("Port: %s", cfg.Port)
	log.Printf("Redis: %s:%s", cfg.RedisHost, cfg.RedisPort)
	log.Printf("Kafka Brokers: %v", cfg.KafkaBrokers)
	log.Printf("CORS Origins: %s", cfg.GetCORSOrigins())

	// Initialize components
	redisClient := redis.NewRedisClient(cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword)

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	} else {
		log.Println("Redis connection successful")
	}

	// Create WebSocket manager with producer
	kafkaBroker := strings.Join(cfg.KafkaBrokers, ",")
	kafkaProducer := kafka.NewKafkaProducer(kafkaBroker, "chat-messages")
	wsManager := delivery.NewWSManager(kafkaProducer, redisClient)

	// Setup Kafka consumer for multi-topic support
	kafkaTopics := []string{"chat-messages", "typing-indicators", "connection-status"}
	kafkaConsumer := kafka.NewKafkaConsumer(
		cfg.KafkaBrokers,
		"livechat-ws-group",
		kafkaTopics,
		wsManager,
	)

	// Create server with configuration
	server := delivery.NewServer(cfg, kafkaConsumer, redisClient, wsManager)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
		if err := kafkaConsumer.Close(); err != nil {
			log.Printf("Error closing Kafka consumer: %v", err)
		}
		if err := kafkaProducer.Close(); err != nil {
			log.Printf("Error closing Kafka producer: %v", err)
		}
		if err := redisClient.Close(); err != nil {
			log.Printf("Error closing Redis client: %v", err)
		}
	}()

	log.Printf("Starting Kafka consumer and WebSocket server...")

	// Start Kafka consumer in background
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Kafka consumer goroutine recovered from panic: %v", r)
			}
		}()

		if err := kafkaConsumer.Start(ctx); err != nil {
			log.Printf("Kafka consumer error: %v", err)
		}
	}()

	// Start server (blocking)
	log.Fatal(server.Start())
}
