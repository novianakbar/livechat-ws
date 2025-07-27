package config

import (
	"os"
	"strings"
)

type Config struct {
	Port             string
	AllowedOrigins   []string
	AllowCredentials bool
	RedisHost        string
	RedisPort        string
	RedisPassword    string
	KafkaBrokers     []string
	Environment      string
}

func LoadConfig() *Config {
	// Get allowed origins from environment variable
	allowedOrigins := []string{"*"} // Default to allow all origins
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		allowedOrigins = strings.Split(origins, ",")
		// Trim whitespace from each origin
		for i, origin := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(origin)
		}
	}

	// Get Kafka brokers
	kafkaBrokers := []string{"localhost:9092"} // Default
	if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
		kafkaBrokers = strings.Split(brokers, ",")
		for i, broker := range kafkaBrokers {
			kafkaBrokers[i] = strings.TrimSpace(broker)
		}
	}

	return &Config{
		Port:             getEnv("PORT", "8082"),
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: getEnv("ALLOW_CREDENTIALS", "false") == "true",
		RedisHost:        getEnv("REDIS_HOST", "localhost"),
		RedisPort:        getEnv("REDIS_PORT", "6379"),
		RedisPassword:    getEnv("REDIS_PASSWORD", ""),
		KafkaBrokers:     kafkaBrokers,
		Environment:      getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetCORSOrigins returns CORS origins as a comma-separated string
func (c *Config) GetCORSOrigins() string {
	if c.Environment == "production" && len(c.AllowedOrigins) > 0 && c.AllowedOrigins[0] != "*" {
		return strings.Join(c.AllowedOrigins, ",")
	}
	return "*"
}

// IsDevelopment returns true if environment is development
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if environment is production
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}
