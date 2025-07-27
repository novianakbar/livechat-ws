<!-- Use this file to provide workspace-specific custom instructions to Copilot. For more details, visit https://code.visualstudio.com/docs/copilot/copilot-customization#_use-a-githubcopilotinstructionsmd-file -->

Project: livechat-ws (WebSocket server for scalable livechat)
- Clean architecture (cmd, internal/delivery, internal/usecase, internal/domain, internal/infrastructure)
- WebSocket server terpisah dari backend utama
- Integrasi Kafka (untuk pesan chat) dan Redis (untuk status koneksi session)
- Endpoint REST untuk query status koneksi session
- Dokumentasi penggunaan dan integrasi
