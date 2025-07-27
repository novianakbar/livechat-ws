package delivery

import (
	"context"
	"log"
	"sync"
	"time"

	"livechat-ws/internal/domain"
	"livechat-ws/internal/infrastructure/kafka"
	"livechat-ws/internal/infrastructure/redis"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

type WSConnection struct {
	Conn      *websocket.Conn
	UserID    string
	UserType  string
	SessionID string
	writeMux  sync.Mutex // Mutex untuk mencegah concurrent write
}

type WSManager struct {
	kafkaProducer *kafka.KafkaProducer
	redisClient   *redis.RedisClient
	// Store active connections by session ID
	connections map[string][]*WSConnection
	mutex       sync.RWMutex
}

func NewWSManager(kafkaProducer *kafka.KafkaProducer, redisClient *redis.RedisClient) *WSManager {
	return &WSManager{
		kafkaProducer: kafkaProducer,
		redisClient:   redisClient,
		connections:   make(map[string][]*WSConnection),
	}
}

func (w *WSManager) addConnection(sessionID string, conn *WSConnection) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if _, exists := w.connections[sessionID]; !exists {
		w.connections[sessionID] = make([]*WSConnection, 0)
	}
	w.connections[sessionID] = append(w.connections[sessionID], conn)
	log.Printf("Added connection: %s (%s) to session %s. Total connections: %d",
		conn.UserID, conn.UserType, sessionID, len(w.connections[sessionID]))
}

func (w *WSManager) removeConnection(sessionID, userID string) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if connections, exists := w.connections[sessionID]; exists {
		for i, conn := range connections {
			if conn.UserID == userID {
				// Remove connection from slice
				w.connections[sessionID] = append(connections[:i], connections[i+1:]...)
				log.Printf("Removed connection: %s from session %s. Remaining connections: %d",
					userID, sessionID, len(w.connections[sessionID]))
				break
			}
		}

		// Clean up empty session
		if len(w.connections[sessionID]) == 0 {
			delete(w.connections, sessionID)
			log.Printf("Cleaned up empty session: %s", sessionID)
		}
	}
}

func (w *WSManager) broadcastToSession(sessionID string, message interface{}) {
	w.mutex.RLock()
	connections := make([]*WSConnection, 0)
	if conns, exists := w.connections[sessionID]; exists {
		// Copy slice untuk menghindari race condition
		connections = make([]*WSConnection, len(conns))
		copy(connections, conns)
	}
	w.mutex.RUnlock()

	if len(connections) == 0 {
		log.Printf("No active connections found for session %s", sessionID)
		return
	}

	successCount := 0
	var wg sync.WaitGroup

	// Broadcast ke semua koneksi secara concurrent tapi thread-safe
	for _, conn := range connections {
		wg.Add(1)
		go func(c *WSConnection) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic while broadcasting to user %s: %v", c.UserID, r)
				}
			}()

			if err := c.safeWriteJSON(message); err != nil {
				log.Printf("Failed to send message to client %s: %v", c.UserID, err)
				// Hapus koneksi yang tidak valid
				w.removeConnection(sessionID, c.UserID)
			} else {
				successCount++
			}
		}(conn)
	}

	wg.Wait()
	log.Printf("Broadcasted message to session %s: %d/%d clients received",
		sessionID, successCount, len(connections))
}

func (w *WSManager) HandleConnection(c *websocket.Conn, sessionID, userID, userType string) {
	defer c.Close()

	ctx := context.Background()

	// Validate session ID format
	if _, err := uuid.Parse(sessionID); err != nil {
		log.Printf("Invalid session ID format: %s", sessionID)
		w.sendErrorResponse(c, "Invalid session ID format")
		return
	}

	// Create connection object
	wsConn := &WSConnection{
		Conn:      c,
		UserID:    userID,
		UserType:  userType,
		SessionID: sessionID,
	}

	// Add to connections map
	w.addConnection(sessionID, wsConn)
	defer func() {
		// First: Broadcast disconnect event BEFORE removing user
		log.Printf("User %s (%s) disconnecting from session %s", userID, userType, sessionID)

		// Remove from connections map and Redis
		w.removeConnection(sessionID, userID)

		// Then: Broadcast updated connection status AFTER user removed with context
		w.broadcastConnectionStatusWithContext(sessionID, "user_disconnected", userID)
	}()

	// Add to Redis
	if err := w.redisClient.AddUserToSession(ctx, sessionID, userID, userType); err != nil {
		log.Printf("Failed to add user to Redis session: %v", err)
	}
	defer func() {
		if err := w.redisClient.RemoveUserFromSession(ctx, sessionID, userID, userType); err != nil {
			log.Printf("Failed to remove user from Redis session: %v", err)
		}
	}()

	// Send connection status updates with connect context
	w.broadcastConnectionStatusWithContext(sessionID, "user_connected", userID)

	// Send welcome message
	w.sendWelcomeMessage(c, sessionID, userID, userType)

	log.Printf("WebSocket client connected: %s (%s) to session %s", userID, userType, sessionID)

	// Handle incoming messages
	for {
		var msg domain.WebSocketMessage
		if err := c.ReadJSON(&msg); err != nil {
			log.Printf("WebSocket read error for user %s: %v", userID, err)
			break
		}

		// Process message based on type
		w.handleIncomingMessage(ctx, c, &msg, sessionID, userID, userType)
	}

	log.Printf("WebSocket client disconnected: %s (%s) from session %s", userID, userType, sessionID)
}

func (w *WSManager) sendWelcomeMessage(c *websocket.Conn, sessionID, userID, userType string) {
	response := domain.WebSocketResponse{
		Type:    "connection_established",
		Success: true,
		Data: map[string]interface{}{
			"session_id": sessionID,
			"user_id":    userID,
			"user_type":  userType,
			"timestamp":  time.Now().Format(time.RFC3339),
			"message":    "Successfully connected to chat session",
		},
	}

	// Gunakan direct write karena ini masih dalam setup koneksi
	if err := w.safeWriteToConn(c, response); err != nil {
		log.Printf("Failed to send welcome message: %v", err)
	}
}

func (w *WSManager) sendErrorResponse(c *websocket.Conn, errorMsg string) {
	response := domain.WebSocketResponse{
		Type:    "error",
		Success: false,
		Error:   errorMsg,
	}

	if err := w.safeWriteToConn(c, response); err != nil {
		log.Printf("Failed to send error response: %v", err)
	}
}

// safeWriteToConn menulis ke koneksi WebSocket dengan recovery dari panic
func (w *WSManager) safeWriteToConn(c *websocket.Conn, message interface{}) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in safeWriteToConn: %v", r)
		}
	}()

	return c.WriteJSON(message)
}

func (w *WSManager) handleIncomingMessage(ctx context.Context, c *websocket.Conn, msg *domain.WebSocketMessage, sessionID, userID, userType string) {
	switch msg.Type {
	case "join_session":
		// Send join confirmation
		response := domain.WebSocketResponse{
			Type:    "session_joined",
			Success: true,
			Data: map[string]interface{}{
				"session_id": sessionID,
				"user_id":    userID,
				"user_type":  userType,
				"timestamp":  time.Now().Format(time.RFC3339),
			},
		}
		w.safeWriteToConn(c, response)

	case "typing_start", "agent_typing":
		isTyping := true
		if msg.Data != nil {
			if dataMap, ok := msg.Data.(map[string]interface{}); ok {
				if typingValue, exists := dataMap["is_typing"]; exists {
					if typing, ok := typingValue.(bool); ok {
						isTyping = typing
					}
				}
			}
		}
		w.handleTypingIndicator(ctx, sessionID, userID, userType, isTyping)

	case "typing_stop":
		w.handleTypingIndicator(ctx, sessionID, userID, userType, false)

	case "send_message":
		w.handleSendMessage(c, msg)

	case "ping":
		// Respond to ping with pong
		response := domain.WebSocketResponse{
			Type:    "pong",
			Success: true,
			Data: map[string]interface{}{
				"timestamp": time.Now().Format(time.RFC3339),
			},
		}
		w.safeWriteToConn(c, response)

	default:
		log.Printf("Unknown message type: %s from user %s", msg.Type, userID)
		w.sendErrorResponse(c, "Unknown message type: "+msg.Type)
	}
}

func (w *WSManager) handleTypingIndicator(ctx context.Context, sessionID, userID, userType string, isTyping bool) {
	// Set typing status in Redis
	if err := w.redisClient.SetUserTyping(ctx, sessionID, userID, isTyping); err != nil {
		log.Printf("Failed to set typing status in Redis: %v", err)
	}

	// Broadcast typing status directly to WebSocket clients
	typingWSMessage := domain.WebSocketResponse{
		Type: "typing_indicator",
		Data: map[string]interface{}{
			"session_id":  sessionID,
			"user_id":     userID,
			"sender_type": userType,
			"is_typing":   isTyping,
			"timestamp":   time.Now().Format(time.RFC3339),
		},
	}
	w.broadcastToSession(sessionID, typingWSMessage)

	// Also send typing status via Kafka for other services
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		log.Printf("Invalid session ID format: %v", err)
		return
	}

	typingMsg := domain.TypingMessage{
		Type:      "typing_indicator",
		SessionID: sessionUUID,
		UserID:    userID,
		UserType:  userType,
		IsTyping:  isTyping,
		Timestamp: time.Now(),
	}

	if err := w.kafkaProducer.SendMessage(ctx, typingMsg); err != nil {
		log.Printf("Failed to send typing message to Kafka: %v", err)
		// Don't return error, continue with WebSocket operation
	}
}

func (w *WSManager) handleSendMessage(c *websocket.Conn, msg *domain.WebSocketMessage) {
	// This would typically send message to backend via API
	// For now, just log it and send confirmation
	log.Printf("Message received from %s: %+v", msg.UserID, msg)

	// Send confirmation back to sender
	response := domain.WebSocketResponse{
		Type:    "message_sent",
		Success: true,
		Data: map[string]interface{}{
			"message_id": uuid.New().String(),
			"timestamp":  time.Now().Format(time.RFC3339),
		},
	}

	if err := w.safeWriteToConn(c, response); err != nil {
		log.Printf("Failed to send message confirmation: %v", err)
	}
}

func (w *WSManager) broadcastConnectionStatusWithContext(sessionID, eventType, eventUserID string) {
	ctx := context.Background()

	// Get connection status from Redis
	status, err := w.redisClient.GetSessionUsers(ctx, sessionID)
	if err != nil {
		log.Printf("Failed to get session users: %v", err)
		return
	}

	// Prepare message data
	messageData := map[string]interface{}{
		"session_id":        sessionID,
		"connection_status": status,
		"timestamp":         time.Now().Format(time.RFC3339),
	}

	// Add context if provided
	if eventType != "" {
		messageData["event_type"] = eventType
	}
	if eventUserID != "" {
		messageData["event_user_id"] = eventUserID
	}

	// Broadcast connection status directly to WebSocket clients
	connectionWSMessage := domain.WebSocketResponse{
		Type: "connection_status_update",
		Data: messageData,
	}
	w.broadcastToSession(sessionID, connectionWSMessage)

	// Also send connection status via Kafka for other services
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		log.Printf("Invalid session ID format: %v", err)
		return
	}

	statusMsg := domain.ConnectionStatusMessage{
		Type:             "connection_status",
		SessionID:        sessionUUID,
		ConnectionStatus: status,
		Timestamp:        time.Now(),
	}

	if err := w.kafkaProducer.SendMessage(ctx, statusMsg); err != nil {
		log.Printf("Failed to send connection status to Kafka: %v", err)
		// Don't return error, continue with WebSocket operation
	}
}

// MessageHandler interface implementation for Kafka message processing
func (w *WSManager) HandleNewMessage(msg domain.ChatMessage) {
	// Recovery dari panic untuk mencegah crash service
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in HandleNewMessage: %v", r)
		}
	}()

	sessionID := msg.SessionID.String()
	log.Printf("HandleNewMessage: SessionID=%s, SenderType=%s, Message=%s",
		sessionID, msg.SenderType, msg.Message)

	// Broadcast new message to WebSocket clients in the session
	wsMessage := domain.WebSocketResponse{
		Type: "new_message",
		Data: map[string]interface{}{
			"message_id":   msg.ID.String(),
			"session_id":   sessionID,
			"sender_id":    msg.SenderID,
			"sender_type":  msg.SenderType,
			"message":      msg.Message,
			"message_type": msg.MessageType,
			"attachments":  msg.Attachments,
			"timestamp":    msg.CreatedAt.Format(time.RFC3339),
		},
	}

	w.broadcastToSession(sessionID, wsMessage)
	log.Printf("Broadcasted new message to session %s", sessionID)
}

func (w *WSManager) HandleTypingIndicator(msg domain.TypingMessage) {
	// Recovery dari panic untuk mencegah crash service
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in HandleTypingIndicator: %v", r)
		}
	}()

	sessionID := msg.SessionID.String()

	// Broadcast typing indicator to WebSocket clients
	wsMessage := domain.WebSocketResponse{
		Type: "typing_indicator",
		Data: map[string]interface{}{
			"session_id":  sessionID,
			"user_id":     msg.UserID,
			"sender_type": msg.UserType,
			"is_typing":   msg.IsTyping,
			"timestamp":   msg.Timestamp.Format(time.RFC3339),
		},
	}

	w.broadcastToSession(sessionID, wsMessage)
	log.Printf("Broadcasted typing indicator to session %s: %s is %s",
		sessionID, msg.UserID, map[bool]string{true: "typing", false: "not typing"}[msg.IsTyping])
}

func (w *WSManager) HandleConnectionStatus(msg domain.ConnectionStatusMessage) {
	// Recovery dari panic untuk mencegah crash service
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in HandleConnectionStatus: %v", r)
		}
	}()

	sessionID := msg.SessionID.String()

	// Broadcast connection status to WebSocket clients
	wsMessage := domain.WebSocketResponse{
		Type: "connection_status_update",
		Data: map[string]interface{}{
			"session_id":        sessionID,
			"connection_status": msg.ConnectionStatus,
			"timestamp":         msg.Timestamp.Format(time.RFC3339),
		},
	}

	w.broadcastToSession(sessionID, wsMessage)
	log.Printf("Broadcasted connection status to session %s", sessionID)
}

// GetActiveConnections returns the current active connections for monitoring
func (w *WSManager) GetActiveConnections() map[string]int {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	result := make(map[string]int)
	for sessionID, connections := range w.connections {
		result[sessionID] = len(connections)
	}
	return result
}

// GetSessionConnectionCount returns the number of active connections for a session
func (w *WSManager) GetSessionConnectionCount(sessionID string) int {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	if connections, exists := w.connections[sessionID]; exists {
		return len(connections)
	}
	return 0
}

// safeWriteJSON writes JSON to WebSocket connection with mutex protection and panic recovery
func (conn *WSConnection) safeWriteJSON(message interface{}) error {
	conn.writeMux.Lock()
	defer conn.writeMux.Unlock()

	// Recovery dari panic untuk mencegah crash
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in safeWriteJSON for user %s: %v", conn.UserID, r)
		}
	}()

	return conn.Conn.WriteJSON(message)
}
