# Fix User Disconnect Event Notification

## Problem
Admin tidak menerima notifikasi real-time ketika customer disconnect dari WebSocket. Event `connection_status_update` sudah ada tetapi timing-nya salah.

## Root Cause Analysis

### Previous Flow (Broken)
```go
defer w.removeConnection(sessionID, userID)  // 1. Remove user dari memory
defer w.redisClient.RemoveUserFromSession()  // 2. Remove user dari Redis

// ... message handling loop ...

w.broadcastConnectionStatus(sessionID)  // 3. Broadcast SETELAH user removed
```

**Masalah**: Ketika `broadcastConnectionStatus` dipanggil, user sudah tidak ada di Redis dan memory map, jadi admin mendapat status yang sudah "bersih" tanpa tahu siapa yang baru disconnect.

### Fixed Flow (Working)
```go
defer func() {
    // 1. Log disconnect event
    log.Printf("User %s (%s) disconnecting from session %s", userID, userType, sessionID)
    
    // 2. Remove user dari memory dan Redis
    w.removeConnection(sessionID, userID)
    
    // 3. Broadcast status SETELAH user removed dengan context
    w.broadcastConnectionStatusWithContext(sessionID, "user_disconnected", userID)
}()

// ... message handling loop ...
// Tidak ada broadcast di akhir karena sudah dipindah ke defer
```

## Changes Made

### 1. Enhanced Connection Status Broadcasting

#### New Method: `broadcastConnectionStatusWithContext`
```go
func (w *WSManager) broadcastConnectionStatusWithContext(sessionID, eventType, eventUserID string) {
    // Get latest status from Redis
    status, err := w.redisClient.GetSessionUsers(ctx, sessionID)
    
    messageData := map[string]interface{}{
        "session_id":        sessionID,
        "connection_status": status,
        "timestamp":         time.Now().Format(time.RFC3339),
    }

    // Add context information
    if eventType != "" {
        messageData["event_type"] = eventType
    }
    if eventUserID != "" {
        messageData["event_user_id"] = eventUserID
    }

    // Broadcast to all clients in session
    connectionWSMessage := domain.WebSocketResponse{
        Type: "connection_status_update",
        Data: messageData,
    }
    w.broadcastToSession(sessionID, connectionWSMessage)
}
```

#### Event Types Added:
- `"user_connected"` - Ketika user baru join session
- `"user_disconnected"` - Ketika user leave session

### 2. Fixed Disconnect Timing

#### Before (Broken)
```go
defer w.removeConnection(sessionID, userID)  // Remove first
defer w.redisClient.RemoveUserFromSession()

// ... loop ...

w.broadcastConnectionStatus(sessionID)  // Broadcast after removal = empty status
```

#### After (Fixed)
```go
defer func() {
    w.removeConnection(sessionID, userID)  // Remove user
    w.broadcastConnectionStatusWithContext(sessionID, "user_disconnected", userID)  // Broadcast with context
}()

// ... loop ...
// No broadcast here anymore
```

### 3. Enhanced Connect Event

#### Before
```go
w.broadcastConnectionStatus(sessionID)  // Generic broadcast
```

#### After  
```go
w.broadcastConnectionStatusWithContext(sessionID, "user_connected", userID)  // With context
```

## WebSocket Event Format

### Previous Event (Generic)
```json
{
  "type": "connection_status_update",
  "data": {
    "session_id": "uuid",
    "connection_status": {
      "users": {...},
      "customer_connected": true,
      "agent_connected": false,
      "total_customer": 1,
      "total_agent": 0
    },
    "timestamp": "2025-01-24T10:30:00Z"
  }
}
```

### New Event (With Context)
```json
{
  "type": "connection_status_update", 
  "data": {
    "session_id": "uuid",
    "connection_status": {
      "users": {...},
      "customer_connected": false,  // Now reflects disconnection
      "agent_connected": true,
      "total_customer": 0,          // Updated count
      "total_agent": 1
    },
    "event_type": "user_disconnected",  // NEW: Event context
    "event_user_id": "customer_123",    // NEW: Which user
    "timestamp": "2025-01-24T10:30:00Z"
  }
}
```

## Frontend Implementation

### Admin Dashboard Handler
```javascript
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  
  if (message.type === 'connection_status_update') {
    const { event_type, event_user_id, connection_status } = message.data;
    
    // Update connection status display
    updateConnectionStatus(connection_status);
    
    // Handle specific events
    if (event_type === 'user_disconnected') {
      showNotification(`${event_user_id} has left the session`, 'info');
      
      // Special handling for customer disconnect
      if (!connection_status.customer_connected && connection_status.agent_connected) {
        showAlert('Customer has left the session', 'warning');
      }
    }
    
    if (event_type === 'user_connected') {
      showNotification(`${event_user_id} has joined the session`, 'success');
      
      // Auto-assign agent if customer joins
      if (connection_status.customer_connected && !connection_status.agent_connected) {
        triggerAgentAssignment(message.data.session_id);
      }
    }
  }
};
```

### React Hook Example
```jsx
const useConnectionStatus = (sessionId) => {
  const [status, setStatus] = useState(null);
  const [lastEvent, setLastEvent] = useState(null);

  useEffect(() => {
    const ws = new WebSocket(`ws://localhost:8081/ws/${sessionId}/admin_${adminId}/admin`);
    
    ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      
      if (message.type === 'connection_status_update') {
        setStatus(message.data.connection_status);
        
        if (message.data.event_type) {
          setLastEvent({
            type: message.data.event_type,
            userId: message.data.event_user_id,
            timestamp: message.data.timestamp
          });
        }
      }
    };

    return () => ws.close();
  }, [sessionId]);

  return { status, lastEvent };
};
```

## Testing

### Manual Testing
1. **Connect Customer**: Open browser, connect to WebSocket as customer
2. **Connect Admin**: Open another browser/tab, connect as admin to same session
3. **Verify Connect Event**: Admin should receive `connection_status_update` with `event_type: "user_connected"`
4. **Disconnect Customer**: Close customer browser/tab
5. **Verify Disconnect Event**: Admin should immediately receive `connection_status_update` with `event_type: "user_disconnected"`

### Test Script
```bash
# Terminal 1: Start server
go run ./cmd

# Terminal 2: Connect as customer
websocat ws://localhost:8081/ws/550e8400-e29b-41d4-a716-446655440000/customer_123/customer

# Terminal 3: Connect as admin  
websocat ws://localhost:8081/ws/550e8400-e29b-41d4-a716-446655440000/admin_456/admin

# Terminal 4: Monitor REST API
watch -n 1 "curl -s http://localhost:8082/api/session/550e8400-e29b-41d4-a716-446655440000/connection-status | jq"
```

### Expected Logs
```
[Connect] WebSocket client connected: customer_123 (customer) to session 550e8400-e29b-41d4-a716-446655440000
[Connect] Broadcasted connection status to session 550e8400-e29b-41d4-a716-446655440000: 2/2 clients received

[Disconnect] User customer_123 (customer) disconnecting from session 550e8400-e29b-41d4-a716-446655440000  
[Disconnect] Removed connection: customer_123 from session 550e8400-e29b-41d4-a716-446655440000. Remaining connections: 1
[Disconnect] Broadcasted connection status to session 550e8400-e29b-41d4-a716-446655440000: 1/1 clients received
[Disconnect] WebSocket client disconnected: customer_123 (customer) from session 550e8400-e29b-41d4-a716-446655440000
```

## Benefits

### 1. Real-time Admin Notifications
- ✅ Admin langsung tahu ketika customer disconnect
- ✅ Admin bisa respond lebih cepat untuk re-engage customer
- ✅ Context event membantu admin understand what happened

### 2. Better User Experience
- ✅ No missed disconnect events
- ✅ More informative notifications
- ✅ Proper timing ensures accurate status

### 3. Improved Monitoring
- ✅ Event-driven notifications
- ✅ Detailed logging for debugging
- ✅ Context information for analytics

## Backward Compatibility

Semua perubahan backward compatible:
- ✅ Event `connection_status_update` masih sama
- ✅ Data structure tidak berubah, hanya ditambah field optional
- ✅ Existing frontend code tetap berfungsi
- ✅ New fields (`event_type`, `event_user_id`) optional

## Future Enhancements

1. **Disconnect Reason**: Add reason for disconnect (network error, manual close, timeout)
2. **Reconnection Detection**: Detect when same user reconnects quickly
3. **Presence Heartbeat**: Add heartbeat mechanism for better disconnect detection
4. **Event History**: Store recent connect/disconnect events for debugging
