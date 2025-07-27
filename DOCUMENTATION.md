# Dokumentasi livechat-ws

## Deskripsi
WebSocket server terpisah untuk livechat scalable. Menggunakan Kafka untuk broadcast pesan chat dan Redis untuk status koneksi session.

## Cara Menjalankan
1. Pastikan Kafka dan Redis sudah berjalan.
2. Copy `.env.example` ke `.env` dan sesuaikan konfigurasi.
3. Jalankan: `go run ./cmd`

## Cara Integrasi
- **livechat-be** publish pesan chat ke Kafka topic yang sama.
- **Client/agent** connect ke WebSocket server ini: `ws://<host>:8081/ws/{session_id}/{user_id}/{user_type}`
- Untuk cek siapa yang online di session: GET `http://<host>:8082/api/session/{session_id}/connection-status`

## Struktur Clean Architecture
- `cmd/` : entrypoint
- `internal/delivery/` : handler WebSocket, REST, Kafka
- `internal/usecase/` : logic broadcast
- `internal/domain/` : entity/event
- `internal/infrastructure/redis/` : Redis adapter
- `internal/infrastructure/kafka/` : Kafka adapter

## Skema Redis
- Setiap join/leave session, update Redis Set: `session:{session_id}:{user_type}`
- Untuk query status, ambil semua member dari Set tersebut.

## Skema Kafka
- Subscribe ke topic pesan chat, broadcast ke client WebSocket sesuai session_id.

## Catatan
- Siap di-scale-out (multi-instance)
- Status koneksi session konsisten di seluruh cluster
