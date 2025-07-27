# Dokumentasi Connection Status - LiveChat WebSocket Server

## Overview
Fitur Connection Status memungkinkan monitoring real-time status koneksi user dalam setiap chat session. Sistem ini menggunakan Redis untuk persistence dan WebSocket untuk real-time updates.

## Arsitektur

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│  Client/UI  │◄──►│ WebSocket    │◄──►│ Redis Store │
│  (Admin)    │    │ Server       │    │             │
└─────────────┘    └──────────────┘    └─────────────┘
                           │
                           ▼
                   ┌──────────────┐
                   │ Kafka Queue  │
                   │ (Optional)   │
                   └──────────────┘
```

## Komponen Utama

### 1. Redis Storage
- **Key Pattern**: `session:{session_id}:users`
- **Data Structure**: Hash Map
- **Content**: User info dengan timestamp join

```json
{
  "user_123": {
    "user_id": "user_123",
    "user_type": "customer",
    "joined_at": "2025-01-24T10:30:00Z"
  },
  "agent_456": {
    "user_id": "agent_456", 
    "user_type": "agent",
    "joined_at": "2025-01-24T10:35:00Z"
  }
}
```

### 2. WebSocket Events
- **Event Type**: `connection_status_update`
- **Trigger**: User join/leave session
- **Broadcast**: Semua client dalam session

### 3. REST API
- **Endpoint**: `GET /api/session/{session_id}/connection-status`
- **Purpose**: Query status untuk admin dashboard

## API Reference

### REST Endpoints

#### Get Session Connection Status
```http
GET /api/session/{session_id}/connection-status
```

**Parameters:**
- `session_id` (path) - UUID session yang ingin dicek

**Response Success (200):**
```json
{
  "success": true,
  "message": "Connection status retrieved successfully",
  "data": {
    "users": {
      "user_123": {
        "user_id": "user_123",
        "user_type": "customer",
        "joined_at": "2025-01-24T10:30:00Z"
      },
      "agent_456": {
        "user_id": "agent_456",
        "user_type": "agent", 
        "joined_at": "2025-01-24T10:35:00Z"
      }
    },
    "customer_connected": true,
    "agent_connected": true,
    "total_customer": 1,
    "total_agent": 1
  }
}
```

**Response Error (400):**
```json
{
  "success": false,
  "message": "Invalid session ID",
  "error": "invalid UUID format"
}
```

**Response Error (500):**
```json
{
  "success": false,
  "message": "Failed to get connection status",
  "error": "redis connection failed"
}
```

### WebSocket Events

#### Connection Status Update
**Event Type:** `connection_status_update`

**Payload:**
```json
{
  "type": "connection_status_update",
  "data": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "connection_status": {
      "users": {
        "user_123": {
          "user_id": "user_123",
          "user_type": "customer",
          "joined_at": "2025-01-24T10:30:00Z"
        }
      },
      "customer_connected": true,
      "agent_connected": false,
      "total_customer": 1,
      "total_agent": 0
    },
    "timestamp": "2025-01-24T10:30:15Z"
  }
}
```

## Implementasi Client

### 1. Admin Dashboard (REST API)

#### JavaScript/Frontend
```javascript
class ConnectionStatusAPI {
  constructor(baseURL) {
    this.baseURL = baseURL;
  }

  async getSessionStatus(sessionId) {
    try {
      const response = await fetch(
        `${this.baseURL}/api/session/${sessionId}/connection-status`
      );
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      
      return await response.json();
    } catch (error) {
      console.error('Failed to get session status:', error);
      throw error;
    }
  }

  async getAllActiveSessions() {
    // Implementasi untuk mendapatkan semua session aktif
    // (perlu endpoint tambahan)
  }
}

// Usage
const api = new ConnectionStatusAPI('http://localhost:8082');

// Poll session status setiap 5 detik
setInterval(async () => {
  try {
    const status = await api.getSessionStatus('session_123');
    updateDashboard(status.data);
  } catch (error) {
    console.error('Status update failed:', error);
  }
}, 5000);

function updateDashboard(connectionStatus) {
  const { customer_connected, agent_connected, total_customer, total_agent } = connectionStatus;
  
  document.getElementById('customer-status').textContent = 
    customer_connected ? `${total_customer} Connected` : 'Offline';
  document.getElementById('agent-status').textContent = 
    agent_connected ? `${total_agent} Connected` : 'Offline';
    
  // Update user list
  const userList = document.getElementById('user-list');
  userList.innerHTML = '';
  
  Object.values(connectionStatus.users).forEach(user => {
    const userElement = document.createElement('div');
    userElement.innerHTML = `
      <div class="user-item">
        <span class="user-id">${user.user_id}</span>
        <span class="user-type badge ${user.user_type}">${user.user_type}</span>
        <span class="join-time">${new Date(user.joined_at).toLocaleTimeString()}</span>
      </div>
    `;
    userList.appendChild(userElement);
  });
}
```

#### React Component
```jsx
import { useState, useEffect } from 'react';

const SessionStatusMonitor = ({ sessionId }) => {
  const [status, setStatus] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const response = await fetch(`/api/session/${sessionId}/connection-status`);
        const data = await response.json();
        
        if (data.success) {
          setStatus(data.data);
          setError(null);
        } else {
          setError(data.message);
        }
      } catch (err) {
        setError('Failed to fetch status');
      } finally {
        setLoading(false);
      }
    };

    fetchStatus();
    const interval = setInterval(fetchStatus, 5000);
    
    return () => clearInterval(interval);
  }, [sessionId]);

  if (loading) return <div>Loading...</div>;
  if (error) return <div className="error">Error: {error}</div>;

  return (
    <div className="session-status">
      <h3>Session {sessionId}</h3>
      
      <div className="status-summary">
        <div className={`status-item ${status.customer_connected ? 'online' : 'offline'}`}>
          <span>Customers: {status.total_customer}</span>
        </div>
        <div className={`status-item ${status.agent_connected ? 'online' : 'offline'}`}>
          <span>Agents: {status.total_agent}</span>
        </div>
      </div>

      <div className="user-list">
        {Object.values(status.users).map(user => (
          <div key={user.user_id} className="user-item">
            <span className="user-id">{user.user_id}</span>
            <span className={`user-type ${user.user_type}`}>{user.user_type}</span>
            <span className="join-time">
              {new Date(user.joined_at).toLocaleTimeString()}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
};

export default SessionStatusMonitor;
```

### 2. Real-time Updates (WebSocket)

#### JavaScript WebSocket Client
```javascript
class LiveChatStatusMonitor {
  constructor(wsURL, sessionId, userId, userType) {
    this.wsURL = wsURL;
    this.sessionId = sessionId;
    this.userId = userId;
    this.userType = userType;
    this.ws = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
  }

  connect() {
    const wsEndpoint = `${this.wsURL}/ws/${this.sessionId}/${this.userId}/${this.userType}`;
    this.ws = new WebSocket(wsEndpoint);

    this.ws.onopen = () => {
      console.log('WebSocket connected');
      this.reconnectAttempts = 0;
    };

    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      this.handleMessage(message);
    };

    this.ws.onclose = () => {
      console.log('WebSocket disconnected');
      this.attemptReconnect();
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
  }

  handleMessage(message) {
    switch (message.type) {
      case 'connection_status_update':
        this.onConnectionStatusUpdate(message.data);
        break;
      case 'connection_established':
        this.onConnectionEstablished(message.data);
        break;
      case 'typing_indicator':
        this.onTypingIndicator(message.data);
        break;
      default:
        console.log('Unknown message type:', message.type);
    }
  }

  onConnectionStatusUpdate(data) {
    const { connection_status } = data;
    
    // Update UI dengan status koneksi terbaru
    this.updateConnectionUI(connection_status);
    
    // Trigger custom events
    document.dispatchEvent(new CustomEvent('connectionStatusUpdate', {
      detail: { sessionId: data.session_id, status: connection_status }
    }));
  }

  updateConnectionUI(status) {
    // Update customer status indicator
    const customerIndicator = document.getElementById('customer-indicator');
    if (customerIndicator) {
      customerIndicator.className = status.customer_connected ? 'online' : 'offline';
      customerIndicator.textContent = `${status.total_customer} customer(s)`;
    }

    // Update agent status indicator  
    const agentIndicator = document.getElementById('agent-indicator');
    if (agentIndicator) {
      agentIndicator.className = status.agent_connected ? 'online' : 'offline';
      agentIndicator.textContent = `${status.total_agent} agent(s)`;
    }

    // Update detailed user list
    this.updateUserList(status.users);
  }

  updateUserList(users) {
    const userList = document.getElementById('connected-users');
    if (!userList) return;

    userList.innerHTML = '';
    
    Object.values(users).forEach(user => {
      const userElement = document.createElement('div');
      userElement.className = `user-item ${user.user_type}`;
      userElement.innerHTML = `
        <div class="user-avatar">${user.user_id.charAt(0).toUpperCase()}</div>
        <div class="user-info">
          <div class="user-name">${user.user_id}</div>
          <div class="user-type">${user.user_type}</div>
          <div class="join-time">Joined: ${new Date(user.joined_at).toLocaleTimeString()}</div>
        </div>
      `;
      userList.appendChild(userElement);
    });
  }

  attemptReconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      const delay = Math.pow(2, this.reconnectAttempts) * 1000; // Exponential backoff
      
      console.log(`Attempting to reconnect in ${delay}ms... (${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
      
      setTimeout(() => {
        this.connect();
      }, delay);
    } else {
      console.error('Max reconnection attempts reached');
    }
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
    }
  }
}

// Usage
const monitor = new LiveChatStatusMonitor(
  'ws://localhost:8081',
  'session_123',
  'admin_456', 
  'admin'
);

monitor.connect();

// Listen for connection status updates
document.addEventListener('connectionStatusUpdate', (event) => {
  const { sessionId, status } = event.detail;
  console.log(`Session ${sessionId} status updated:`, status);
  
  // Trigger notifications if needed
  if (!status.agent_connected && status.customer_connected) {
    showNotification('Customer waiting for agent!', 'warning');
  }
});
```

### 3. Backend Integration (untuk Service Lain)

#### Go Service
```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
)

type ConnectionStatusClient struct {
    baseURL string
    client  *http.Client
}

type ConnectionStatus struct {
    Users             map[string]UserInfo `json:"users"`
    CustomerConnected bool                `json:"customer_connected"`
    AgentConnected    bool                `json:"agent_connected"`
    TotalCustomer     int                 `json:"total_customer"`
    TotalAgent        int                 `json:"total_agent"`
}

type UserInfo struct {
    UserID   string `json:"user_id"`
    UserType string `json:"user_type"`
    JoinedAt string `json:"joined_at"`
}

func NewConnectionStatusClient(baseURL string) *ConnectionStatusClient {
    return &ConnectionStatusClient{
        baseURL: baseURL,
        client:  &http.Client{},
    }
}

func (c *ConnectionStatusClient) GetSessionStatus(sessionID string) (*ConnectionStatus, error) {
    url := fmt.Sprintf("%s/api/session/%s/connection-status", c.baseURL, sessionID)
    
    resp, err := c.client.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var response struct {
        Success bool             `json:"success"`
        Message string           `json:"message"`
        Data    ConnectionStatus `json:"data"`
        Error   string           `json:"error,omitempty"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return nil, err
    }

    if !response.Success {
        return nil, fmt.Errorf("API error: %s", response.Message)
    }

    return &response.Data, nil
}

// Usage
func main() {
    client := NewConnectionStatusClient("http://localhost:8082")
    
    status, err := client.GetSessionStatus("session_123")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Session has %d customers and %d agents\n", 
        status.TotalCustomer, status.TotalAgent)
        
    if status.CustomerConnected && !status.AgentConnected {
        // Trigger auto-assign agent logic
        assignAvailableAgent("session_123")
    }
}
```

## Use Cases

### 1. Admin Dashboard
- **Monitoring**: Real-time overview semua session aktif
- **Agent Management**: Assign agent ke session yang butuh bantuan
- **Analytics**: Track response time dan customer satisfaction

### 2. Agent Application  
- **Queue Management**: Lihat antrian customer yang waiting
- **Session Overview**: Check siapa saja yang ada di session
- **Availability Status**: Update agent availability

### 3. Customer Support
- **Auto-routing**: Route customer ke agent yang available
- **Wait Time Estimation**: Berikan estimasi waktu tunggu
- **Escalation**: Auto-escalate jika customer tunggu terlalu lama

### 4. System Integration
- **CRM Integration**: Sync status dengan CRM system
- **Analytics**: Send metrics ke analytics platform
- **Notifications**: Trigger email/SMS notifications

## Monitoring & Troubleshooting

### Health Check
```bash
# Check server health
curl http://localhost:8082/health

# Check specific session
curl http://localhost:8082/api/session/550e8400-e29b-41d4-a716-446655440000/connection-status
```

### Redis Monitoring
```bash
# Check Redis keys
redis-cli KEYS "session:*:users"

# Check specific session
redis-cli HGETALL "session:550e8400-e29b-41d4-a716-446655440000:users"

# Monitor Redis operations
redis-cli MONITOR
```

### Common Issues

#### 1. Stale Connections
**Problem**: User terlihat online padahal sudah disconnect
**Solution**: Implement heartbeat mechanism

#### 2. Redis Memory Usage
**Problem**: Memory usage terus naik
**Solution**: Set TTL untuk session keys

#### 3. WebSocket Connection Drops
**Problem**: Frequent disconnections
**Solution**: Implement reconnection with exponential backoff

## Performance Considerations

### 1. Scaling
- **Redis Clustering**: Untuk handle banyak sessions
- **Load Balancing**: Multiple WebSocket server instances
- **CDN**: Static assets untuk dashboard

### 2. Optimization
- **Connection Pooling**: Redis connection pool
- **Caching**: Cache frequently accessed sessions
- **Batch Updates**: Batch Redis operations

### 3. Limits
- **Max Connections**: Limit per session (default: unlimited)
- **Rate Limiting**: API rate limiting untuk prevent abuse
- **Memory Limits**: Set appropriate Redis memory limits

## Security

### 1. Authentication
- Validate user_id dan session_id
- Check user permissions untuk access session
- Rate limiting untuk prevent DoS

### 2. Authorization  
- Admin hanya bisa access session yang authorized
- Agent hanya bisa monitor assigned sessions
- Customer hanya bisa access own session

### 3. Data Privacy
- Mask sensitive user data dalam logs
- Encrypt Redis data jika needed
- Audit log untuk admin access

---

## Changelog

**v1.0.0** (2025-01-24)
- Initial implementation
- Basic connection status tracking
- REST API dan WebSocket events
- Redis persistence

**Future Enhancements**
- [ ] Session history tracking
- [ ] Advanced analytics
- [ ] Mobile push notifications
- [ ] Multi-tenant support
