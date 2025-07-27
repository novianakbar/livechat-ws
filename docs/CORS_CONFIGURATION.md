# CORS Configuration Guide - LiveChat WebSocket Server

## Overview
Cross-Origin Resource Sharing (CORS) telah dikonfigurasi dengan best practices untuk mendukung integrasi frontend yang aman dan fleksibel.

## Konfigurasi CORS

### Environment Variables

```bash
# CORS Origins - specify allowed origins
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3001,https://yourdomain.com

# Allow credentials (cookies, authorization headers)
ALLOW_CREDENTIALS=false

# Environment (affects CORS behavior)
ENVIRONMENT=development
```

### Development vs Production

#### Development Mode (`ENVIRONMENT=development`)
- **AllowOrigins**: `*` (wildcard - semua origins diperbolehkan)
- **AllowCredentials**: `false` (tidak bisa true dengan wildcard origin)
- **Logging**: Debug mode dengan detailed CORS info

#### Production Mode (`ENVIRONMENT=production`)
- **AllowOrigins**: Hanya origins yang disebutkan di `ALLOWED_ORIGINS`
- **AllowCredentials**: Sesuai setting `ALLOW_CREDENTIALS`
- **Security**: Strict mode untuk production

### Headers Yang Diperbolehkan

#### Request Headers
- `Origin`
- `Content-Type`
- `Accept`
- `Authorization`
- `X-Requested-With`
- `Access-Control-Request-Method`
- `Access-Control-Request-Headers`

#### Response Headers Exposed
- `Content-Length`
- `Access-Control-Allow-Origin`
- `Access-Control-Allow-Headers`
- `Content-Type`

#### HTTP Methods
- `GET`
- `POST`
- `HEAD`
- `PUT`
- `DELETE`
- `PATCH`
- `OPTIONS`

## Contoh Konfigurasi

### 1. Development Environment
```bash
# .env.development
ENVIRONMENT=development
PORT=8082
ALLOWED_ORIGINS=*
ALLOW_CREDENTIALS=false
```

### 2. Staging Environment  
```bash
# .env.staging
ENVIRONMENT=production
PORT=8082
ALLOWED_ORIGINS=https://staging.yourapp.com,https://admin-staging.yourapp.com
ALLOW_CREDENTIALS=true
```

### 3. Production Environment
```bash
# .env.production
ENVIRONMENT=production
PORT=8082
ALLOWED_ORIGINS=https://yourapp.com,https://admin.yourapp.com,https://mobile.yourapp.com
ALLOW_CREDENTIALS=true
```

## Frontend Integration

### JavaScript/Fetch API
```javascript
// Development
const response = await fetch('http://localhost:8082/api/session/123/connection-status', {
  method: 'GET',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer your-token'
  },
  credentials: 'omit' // or 'include' if ALLOW_CREDENTIALS=true
});

// Production dengan credentials
const response = await fetch('https://ws-api.yourapp.com/api/session/123/connection-status', {
  method: 'GET',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer your-token'
  },
  credentials: 'include' // Kirim cookies dan auth headers
});
```

### Axios Configuration
```javascript
// axios-config.js
import axios from 'axios';

const apiClient = axios.create({
  baseURL: process.env.NODE_ENV === 'development' 
    ? 'http://localhost:8082'
    : 'https://ws-api.yourapp.com',
  withCredentials: true, // Kirim cookies
  headers: {
    'Content-Type': 'application/json',
  }
});

// Request interceptor untuk auth token
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('authToken');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

export default apiClient;
```

### React Hook
```jsx
import { useState, useEffect } from 'react';
import apiClient from './axios-config';

const useConnectionStatus = (sessionId) => {
  const [status, setStatus] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const response = await apiClient.get(`/api/session/${sessionId}/connection-status`);
        setStatus(response.data.data);
        setError(null);
      } catch (err) {
        setError(err.response?.data?.message || 'Failed to fetch status');
      } finally {
        setLoading(false);
      }
    };

    fetchStatus();
    const interval = setInterval(fetchStatus, 5000);
    
    return () => clearInterval(interval);
  }, [sessionId]);

  return { status, loading, error };
};

export default useConnectionStatus;
```

### WebSocket CORS
WebSocket connections tidak menggunakan CORS dalam cara yang sama seperti HTTP requests, tetapi browser akan melakukan origin check.

```javascript
// WebSocket connection
const ws = new WebSocket('ws://localhost:8081/ws/session_123/user_456/customer');

ws.onopen = () => {
  console.log('WebSocket connected');
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
  // Biasanya karena CORS atau network issues
};
```

## Testing CORS

### Manual Testing dengan curl
```bash
# Test preflight request
curl -X OPTIONS http://localhost:8082/api/session/123/connection-status \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: GET" \
  -H "Access-Control-Request-Headers: Content-Type,Authorization" \
  -v

# Test actual request
curl -X GET http://localhost:8082/api/session/123/connection-status \
  -H "Origin: http://localhost:3000" \
  -H "Content-Type: application/json" \
  -v
```

### Browser DevTools
1. Open Developer Tools (F12)
2. Go to Network tab
3. Make a request to the API
4. Check response headers:
   - `Access-Control-Allow-Origin`
   - `Access-Control-Allow-Methods`
   - `Access-Control-Allow-Headers`

### Health Check dengan CORS Info
```bash
curl http://localhost:8082/health
```

Response akan include informasi CORS:
```json
{
  "status": "ok",
  "message": "LiveChat WebSocket server is running",
  "port": "8082",
  "environment": "development",
  "cors_origins": "*"
}
```

## Common CORS Issues & Solutions

### 1. "CORS policy: No 'Access-Control-Allow-Origin' header"
**Penyebab**: Origin tidak ada dalam `ALLOWED_ORIGINS`
**Solusi**: 
- Development: Set `ENVIRONMENT=development` untuk wildcard
- Production: Tambahkan domain ke `ALLOWED_ORIGINS`

### 2. "CORS policy: Credential is not supported if the CORS header 'Access-Control-Allow-Origin' is '*'"
**Penyebab**: `withCredentials: true` dengan wildcard origin
**Solusi**: 
- Set specific origins di production mode
- Atau set `ALLOW_CREDENTIALS=false`

### 3. "CORS policy: Request header field Authorization is not allowed"
**Penyebab**: Header `Authorization` tidak ada dalam allowed headers
**Solusi**: Sudah included dalam konfigurasi default

### 4. WebSocket Connection Failed
**Penyebab**: Browser blocking WebSocket karena origin mismatch
**Solusi**: 
- Pastikan WebSocket URL dan API URL sama-sama di allowed origins
- Gunakan same protocol (http/https, ws/wss)

## Security Best Practices

### 1. Production Configuration
```bash
# Jangan gunakan wildcard di production
ALLOWED_ORIGINS=https://yourapp.com,https://admin.yourapp.com

# Hati-hati dengan credentials
ALLOW_CREDENTIALS=true  # Hanya jika benar-benar dibutuhkan

# Selalu set environment dengan benar
ENVIRONMENT=production
```

### 2. HTTPS Only in Production
```bash
# Pastikan menggunakan HTTPS
ALLOWED_ORIGINS=https://yourapp.com  # Bukan http://

# WebSocket juga menggunakan WSS
# ws://  -> tidak aman
# wss:// -> aman
```

### 3. Minimal Permissions
- Hanya allow origins yang benar-benar dibutuhkan
- Hanya allow methods yang digunakan
- Hanya set credentials=true jika dibutuhkan

### 4. Regular Security Audit
- Review `ALLOWED_ORIGINS` secara berkala
- Monitor CORS errors di logs
- Test CORS configuration sebelum deploy

## Troubleshooting Checklist

- [ ] Check `ENVIRONMENT` variable (development/production)
- [ ] Verify `ALLOWED_ORIGINS` contains your frontend domain
- [ ] Ensure protocol match (http/https)
- [ ] Check `ALLOW_CREDENTIALS` setting
- [ ] Test with browser DevTools Network tab
- [ ] Verify server logs for CORS configuration
- [ ] Test both API endpoints dan WebSocket connections

---

## Support

Jika masih ada issues dengan CORS:
1. Check server logs saat request gagal
2. Use browser DevTools untuk detail error
3. Test dengan `curl` untuk isolate masalah
4. Verify environment variables dengan `/health` endpoint
