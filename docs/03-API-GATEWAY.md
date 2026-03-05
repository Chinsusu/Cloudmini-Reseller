# API Gateway — Service Design

**Document ID**: PVP-DOC-003  
**Version**: 1.0.0  
**Service**: api-gateway  
**Port**: 8080  

---

## 1. Responsibilities

- JWT validation và injection user claims vào request headers
- Request ID generation (`X-Request-ID`) cho toàn bộ request lifecycle
- Rate limiting theo user tier
- Route forwarding đến downstream services
- TLS termination (sau Cloudflare/Nginx)
- Access logging

## 2. Middleware Chain

```
Request
  │
  ├─ 1. RecoveryMiddleware      ← panic recovery
  ├─ 2. RequestIDMiddleware     ← inject X-Request-ID
  ├─ 3. LoggingMiddleware       ← log request/response
  ├─ 4. CORSMiddleware          ← CORS headers
  ├─ 5. RateLimitMiddleware     ← per-user, per-tier sliding window
  ├─ 6. AuthMiddleware          ← JWT validate, extract claims
  ├─ 7. RBACMiddleware          ← check role permissions
  └─ 8. ProxyMiddleware         ← forward to service
```

## 3. Rate Limit Tiers

| Tier | Requests/minute | Requests/hour |
|---|---|---|
| `anonymous` | 20 | 200 |
| `user` | 120 | 3,000 |
| `reseller` | 500 | 10,000 |
| `admin` | 2,000 | Unlimited |

Rate limit state lưu trong Redis với sliding window algorithm.

## 4. Route Table

| Method | Path Pattern | Downstream | Auth Required |
|---|---|---|---|
| POST | `/api/v1/auth/*` | user-service | No |
| GET/PUT | `/api/v1/users/*` | user-service | Yes |
| `*` | `/api/v1/proxy/*` | proxy-service | Yes |
| `*` | `/api/v1/vps/*` | vps-service | Yes |
| `*` | `/api/v1/billing/*` | billing-service | Yes |
| `*` | `/api/v1/logs/*` | log-service | Yes |
| `*` | `/api/v1/admin/*` | admin routes | Yes + admin role |
| `*` | `/api/v1/reseller/*` | reseller-service | Yes + reseller role |
| GET | `/ws/events` | log-service (WebSocket) | Yes |

## 5. Headers Injected to Downstream

```
X-Request-ID:   {uuid}
X-User-ID:      {user_id from JWT}
X-User-Role:    {role from JWT}
X-Reseller-ID:  {reseller_id or empty}
X-Real-IP:      {client IP}
```

## 6. Error Response Format

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests. Retry after 60 seconds.",
    "request_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

## 7. Health Check

```
GET /health
→ 200 { "status": "ok", "uptime": 12345, "version": "1.0.0" }
```
