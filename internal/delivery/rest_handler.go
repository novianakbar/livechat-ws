package delivery

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (s *Server) handleGetSessionConnectionStatus(c *fiber.Ctx) error {
	sessionIDStr := c.Params("session_id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid session ID",
			"error":   err.Error(),
		})
	}

	// Get connection status from Redis
	status, err := s.redis.GetSessionUsers(c.Context(), sessionID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to get connection status",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Connection status retrieved successfully",
		"data":    status,
	})
}
