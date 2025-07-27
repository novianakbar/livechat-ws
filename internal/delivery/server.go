package delivery

import (
	"log"

	"livechat-ws/internal/config"
	"livechat-ws/internal/infrastructure/kafka"
	"livechat-ws/internal/infrastructure/redis"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"
)

type Server struct {
	config        *config.Config
	kafkaConsumer *kafka.KafkaConsumer
	redis         *redis.RedisClient
	wsManager     *WSManager
}

func NewServer(config *config.Config, kafkaConsumer *kafka.KafkaConsumer, redis *redis.RedisClient, wsManager *WSManager) *Server {
	return &Server{
		config:        config,
		kafkaConsumer: kafkaConsumer,
		redis:         redis,
		wsManager:     wsManager,
	}
}

func (s *Server) Start() error {
	app := fiber.New(fiber.Config{
		AppName: "LiveChat WebSocket & REST Server",
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} ${latency}\n",
	}))

	// CORS middleware with best practices
	corsConfig := cors.Config{
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Requested-With,Access-Control-Request-Method,Access-Control-Request-Headers",
		ExposeHeaders:    "Content-Length,Access-Control-Allow-Origin,Access-Control-Allow-Headers,Content-Type",
		AllowCredentials: s.config.AllowCredentials,
		MaxAge:           86400, // 24 hours
	}

	// Set origins based on environment
	if s.config.IsProduction() {
		corsConfig.AllowOrigins = s.config.GetCORSOrigins()
		log.Printf("CORS configured for production with origins: %s", corsConfig.AllowOrigins)
	} else {
		corsConfig.AllowOrigins = "*"
		corsConfig.AllowCredentials = false // Never allow credentials with wildcard origin
		log.Printf("CORS configured for development with wildcard origin")
	}

	app.Use(cors.New(corsConfig))

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":       "ok",
			"message":      "LiveChat WebSocket server is running",
			"port":         s.config.Port,
			"environment":  s.config.Environment,
			"cors_origins": s.config.GetCORSOrigins(),
		})
	})

	// REST API routes
	api := app.Group("/api")
	api.Get("/session/:session_id/connection-status", s.handleGetSessionConnectionStatus)

	// WebSocket middleware
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// WebSocket route
	app.Get("/ws/:session_id/:user_id/:user_type", websocket.New(func(c *websocket.Conn) {
		params := []string{c.Params("session_id"), c.Params("user_id"), c.Params("user_type")}
		sessionID, userID, userType := params[0], params[1], params[2]

		// Handle connection through WebSocket manager
		s.wsManager.HandleConnection(c, sessionID, userID, userType)
	}))

	log.Printf("LiveChat server (WebSocket + REST) starting on port %s", s.config.Port)
	return app.Listen(":" + s.config.Port)
}
