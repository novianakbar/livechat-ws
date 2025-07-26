# LiveChat WebSocket Server

WebSocket server terpisah untuk sistem livechat yang scalable. Server ini menggunakan Kafka untuk message broadcasting dan Redis untuk session management.

## 🚀 Features

- **Real-time WebSocket Communication**: Instant messaging dengan typing indicators
- **Connection Status Tracking**: Monitor user connections dalam real-time
- **CORS Support**: Konfigurasi CORS yang fleksibel untuk development dan production
- **Kafka Integration**: Message broadcasting via Kafka topics
- **Redis Session Management**: Persistent session dan connection tracking
- **REST API**: Query connection status via HTTP endpoints
- **Clean Architecture**: Organized codebase dengan separation of concerns
- **Graceful Shutdown**: Proper cleanup saat server shutdown

## 🏗️ Architecture

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   Frontend  │◄──►│   WebSocket  │◄──►│    Redis    │
│   Client    │    │    Server    │    │   Storage   │
└─────────────┘    └──────────────┘    └─────────────┘
                           │
                           ▼
                   ┌──────────────┐
                   │    Kafka     │
                   │   Message    │
                   │    Queue     │
                   └──────────────┘
```

## 📋 Prerequisites

- Go 1.21+
- Redis Server
- Apache Kafka
- (Optional) Docker & Docker Compose

## ⚙️ Configuration

### Environment Variables

Copy `.env.example` to `.env` dan sesuaikan konfigurasi:

```bash
cp .env.example .env
```

### CORS Configuration

Server mendukung konfigurasi CORS yang fleksibel:

```bash
# Development - allow all origins
ENVIRONMENT=development
ALLOWED_ORIGINS=*

# Production - specific origins only
ENVIRONMENT=production
ALLOWED_ORIGINS=https://yourapp.com,https://admin.yourapp.com
ALLOW_CREDENTIALS=true
```

**📖 Dokumentasi CORS lengkap**: [docs/CORS_CONFIGURATION.md](docs/CORS_CONFIGURATION.md)

### Complete Configuration Example

```bash
# Server Configuration
PORT=8082
ENVIRONMENT=development

# CORS Configuration
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3001
ALLOW_CREDENTIALS=false

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# Kafka Configuration  
KAFKA_BROKERS=localhost:9092
```

## 🚀 Quick Start

### Option 1: Using Docker (Recommended)

```bash
# Start Redis dan Kafka
docker-compose up -d

# Build dan run server
go build -o bin/livechat-ws ./cmd
./bin/livechat-ws
```

### Option 2: Manual Setup

1. **Start Redis**:
   ```bash
   redis-server
   ```

2. **Start Kafka**:
   ```bash
   # Start Zookeeper
   bin/zookeeper-server-start.sh config/zookeeper.properties
   
   # Start Kafka
   bin/kafka-server-start.sh config/server.properties
   ```

3. **Run Server**:
   ```bash
   go run ./cmd
   ```

## 📡 API Endpoints

### REST API

#### Health Check
```http
GET /health
```

Response:
```json
{
  "status": "ok",
  "message": "LiveChat WebSocket server is running",
  "port": "8082",
  "environment": "development",
  "cors_origins": "*"
}
```

#### Connection Status
```http
GET /api/session/{session_id}/connection-status
```

Response:
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
      }
    },
    "customer_connected": true,
    "agent_connected": false,
    "total_customer": 1,
    "total_agent": 0
  }
}
```

### WebSocket Connection

```
ws://localhost:8081/ws/{session_id}/{user_id}/{user_type}
```

**Parameters:**
- `session_id`: UUID session chat
- `user_id`: Unique identifier user
- `user_type`: `customer` atau `agent`

**📖 Dokumentasi Connection Status lengkap**: [docs/CONNECTION_STATUS.md](docs/CONNECTION_STATUS.md)

## 💻 Frontend Integration

### JavaScript/Fetch API
```javascript
// API Call dengan CORS
const response = await fetch('http://localhost:8082/api/session/123/connection-status', {
  method: 'GET',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer your-token'
  },
  credentials: 'omit' // atau 'include' jika ALLOW_CREDENTIALS=true
});

const data = await response.json();
console.log(data);
```

### WebSocket Connection
```javascript
const ws = new WebSocket('ws://localhost:8081/ws/session_123/user_456/customer');

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message);
};

ws.send(JSON.stringify({
  type: 'send_message',
  data: {
    message: 'Hello from client'
  }
}));
```

## 🔧 Development

### Build

```bash
# Build binary
go build -o bin/livechat-ws ./cmd

# Run binary
./bin/livechat-ws
```

### Testing

```bash
# Run tests
go test ./...

# Test dengan coverage
go test -cover ./...
```

### Development Mode

Set environment untuk development:

```bash
export ENVIRONMENT=development
export ALLOWED_ORIGINS=*
go run ./cmd
```

## 🚀 Production Deployment

### Environment Configuration

```bash
# Production settings
export ENVIRONMENT=production
export ALLOWED_ORIGINS=https://yourapp.com,https://admin.yourapp.com
export ALLOW_CREDENTIALS=true
export PORT=8082
```

### Security Considerations

1. **CORS Origins**: Jangan gunakan wildcard `*` di production
2. **HTTPS Only**: Gunakan HTTPS untuk production
3. **Credentials**: Set `ALLOW_CREDENTIALS=true` hanya jika dibutuhkan
4. **Firewall**: Restrict access ke Redis dan Kafka

### Docker Production

```dockerfile
# Multi-stage build
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o bin/livechat-ws ./cmd

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/bin/livechat-ws .
CMD ["./livechat-ws"]
```

## 📊 Monitoring

### Health Checks

```bash
# Server health
curl http://localhost:8082/health

# Connection status
curl http://localhost:8082/api/session/YOUR_SESSION_ID/connection-status
```

### Logs

Server menggunakan structured logging:

```
[2025-01-24T10:30:00Z] 200 - GET /health 1.2ms
[2025-01-24T10:30:05Z] WebSocket client connected: user_123 (customer) to session session_456
[2025-01-24T10:30:10Z] Broadcasted message to session session_456: 2/2 clients received
```

## 🛠️ Troubleshooting

### Common Issues

1. **CORS Errors**
   - Check `ALLOWED_ORIGINS` configuration
   - Verify `ENVIRONMENT` setting
   - See [CORS Configuration Guide](docs/CORS_CONFIGURATION.md)

2. **WebSocket Connection Failed**
   - Check server is running on correct port
   - Verify WebSocket URL format
   - Check browser console for detailed errors

3. **Redis Connection Failed**
   - Verify Redis server is running
   - Check `REDIS_HOST` dan `REDIS_PORT`
   - Test Redis connection: `redis-cli ping`

4. **Kafka Consumer Errors**
   - Verify Kafka server is running
   - Check `KAFKA_BROKERS` configuration
   - Create required topics manually if needed

### Debug Mode

Set log level untuk debugging:

```bash
export LOG_LEVEL=debug
go run ./cmd
```

## 📚 Documentation

- [CORS Configuration](docs/CORS_CONFIGURATION.md) - Detailed CORS setup guide
- [Connection Status API](docs/CONNECTION_STATUS.md) - Complete API documentation
- [Architecture Overview](DOCUMENTATION.md) - System architecture dan design

## 🤝 Contributing

1. Fork repository
2. Create feature branch: `git checkout -b feature/new-feature`
3. Commit changes: `git commit -am 'Add new feature'`
4. Push branch: `git push origin feature/new-feature`
5. Submit Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🔗 Related Projects

- **livechat-be**: Backend API untuk chat management
- **livechat-frontend**: Frontend application untuk customers
- **livechat-admin**: Admin dashboard untuk agents

---

**Built with ❤️ using Go, Fiber, Redis, and Kafka**
