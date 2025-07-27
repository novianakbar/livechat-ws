package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func (r *RedisClient) AddUserToSession(ctx context.Context, sessionID, userID, userType string) error {
	key := fmt.Sprintf("session:%s:users", sessionID)
	userInfo := map[string]interface{}{
		"user_id":   userID,
		"user_type": userType,
		"joined_at": time.Now(),
	}

	userJSON, err := json.Marshal(userInfo)
	if err != nil {
		return err
	}

	return r.client.HSet(ctx, key, userID, userJSON).Err()
}

func (r *RedisClient) RemoveUserFromSession(ctx context.Context, sessionID, userID, userType string) error {
	key := fmt.Sprintf("session:%s:users", sessionID)
	return r.client.HDel(ctx, key, userID).Err()
}

func (r *RedisClient) GetSessionUsers(ctx context.Context, sessionID string) (map[string]interface{}, error) {
	key := fmt.Sprintf("session:%s:users", sessionID)
	users, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	customerCount := 0
	agentCount := 0

	for userID, userJSON := range users {
		var userInfo map[string]interface{}
		if err := json.Unmarshal([]byte(userJSON), &userInfo); err != nil {
			continue
		}

		userType, _ := userInfo["user_type"].(string)
		if userType == "customer" {
			customerCount++
		} else if userType == "agent" {
			agentCount++
		}

		result[userID] = userInfo
	}

	return map[string]interface{}{
		"users":              result,
		"customer_connected": customerCount > 0,
		"agent_connected":    agentCount > 0,
		"total_customer":     customerCount,
		"total_agent":        agentCount,
	}, nil
}

func (r *RedisClient) SetUserTyping(ctx context.Context, sessionID, userID string, isTyping bool) error {
	key := fmt.Sprintf("session:%s:typing:%s", sessionID, userID)
	if isTyping {
		return r.client.Set(ctx, key, "true", 30*time.Second).Err()
	} else {
		return r.client.Del(ctx, key).Err()
	}
}

func (r *RedisClient) GetTypingUsers(ctx context.Context, sessionID string) ([]string, error) {
	pattern := fmt.Sprintf("session:%s:typing:*", sessionID)
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	var typingUsers []string
	for _, key := range keys {
		// Extract user ID from key pattern: session:{sessionID}:typing:{userID}
		prefix := fmt.Sprintf("session:%s:typing:", sessionID)
		if len(key) > len(prefix) {
			userID := key[len(prefix):]
			typingUsers = append(typingUsers, userID)
		}
	}

	return typingUsers, nil
}

func (r *RedisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}
