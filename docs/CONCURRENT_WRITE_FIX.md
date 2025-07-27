# Solusi Concurrent WebSocket Write Error

## Masalah
Service livechat-ws mengalami crash dengan error:
```
panic: concurrent write to websocket connection
```

Error ini terjadi karena beberapa goroutine mencoba menulis ke koneksi WebSocket yang sama secara bersamaan, yang tidak diperbolehkan oleh library WebSocket.

## Root Cause
1. **Kafka Consumer** berjalan di goroutine terpisah dan memproses pesan `connection-status`
2. **WebSocket Handler** juga berjalan di goroutine terpisah untuk setiap koneksi
3. Kedua goroutine ini bisa memanggil `broadcastToSession()` secara bersamaan
4. `broadcastToSession()` memanggil `WriteJSON()` pada koneksi WebSocket yang sama
5. Library WebSocket tidak thread-safe untuk operasi write

## Solusi yang Diimplementasi

### 1. Thread-Safe WebSocket Writing
- **Menambahkan mutex per koneksi**: Setiap `WSConnection` sekarang memiliki `writeMux sync.Mutex`
- **Method `safeWriteJSON()`**: Wrapper yang menggunakan mutex untuk memastikan hanya satu goroutine yang menulis pada satu waktu
- **Method `safeWriteToConn()`**: Untuk operasi write langsung ke koneksi

### 2. Improved Broadcasting
- **Copy slice sebelum broadcast**: Menghindari race condition saat iterasi
- **Concurrent broadcasting**: Setiap koneksi di-broadcast secara parallel tapi thread-safe
- **Auto cleanup**: Koneksi yang gagal dikirim otomatis dihapus

### 3. Panic Recovery
- **Per-connection recovery**: Setiap operasi write memiliki panic recovery
- **Per-method recovery**: Semua handler method (`HandleConnectionStatus`, `HandleTypingIndicator`, `HandleNewMessage`) memiliki recovery
- **Goroutine recovery**: Kafka consumer goroutine memiliki recovery
- **Global recovery**: Main application memiliki recovery di level teratas

### 4. Connection Management
- **Graceful error handling**: Koneksi yang error otomatis dihapus dari pool
- **Proper cleanup**: Session kosong otomatis dibersihkan
- **Logging yang lebih baik**: Error tracking yang lebih detail

## Perubahan File

### `/internal/delivery/ws_manager.go`
- Menambahkan `writeMux sync.Mutex` di `WSConnection`
- Method `safeWriteJSON()` untuk thread-safe writing
- Method `safeWriteToConn()` untuk direct writing
- Improved `broadcastToSession()` dengan concurrent tapi safe broadcasting
- Panic recovery di semua handler methods

### `/internal/infrastructure/kafka/consumer.go`
- Panic recovery di `handleMessage()`
- Panic recovery di setiap goroutine consumer

### `/cmd/main.go`
- Global panic recovery di main function
- Panic recovery di Kafka consumer goroutine

## Testing

Untuk memastikan solusi bekerja:

1. **Load Testing**: Test dengan multiple concurrent connections
2. **Message Broadcasting**: Test broadcast ke multiple sessions
3. **Kafka Message Processing**: Test high-volume Kafka messages
4. **Connection Drops**: Test handling koneksi yang putus tiba-tiba

## Best Practices

1. **Selalu gunakan `safeWriteJSON()`** untuk broadcast operations
2. **Gunakan `safeWriteToConn()`** untuk direct write operations
3. **Monitor logs** untuk panic recovery events
4. **Setup proper monitoring** untuk track connection health

## Monitoring

Log messages yang perlu dimonitor:
- `"Recovered from panic in"` - Indicates panic recovery
- `"Failed to send message to client"` - Connection issues
- `"Removed connection"` - Auto cleanup events
- `"Broadcasted message to session"` - Successful broadcasts

## Future Improvements

1. **Connection pooling**: Untuk better resource management
2. **Circuit breaker**: Untuk handle Kafka downtime
3. **Health checks**: Untuk monitor connection status
4. **Metrics**: Untuk performance monitoring
