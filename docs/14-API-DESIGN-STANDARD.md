# API Design Standard — ProxyVPS Platform

**Document ID**: PVP-DOC-014  
**Version**: 1.0.0  

---

## 1. General Conventions

- **Base URL**: `/api/v1/`
- **Protocol**: HTTPS only
- **Content-Type**: `application/json`
- **Charset**: UTF-8
- **Date format**: ISO 8601 — `2025-01-01T00:00:00Z`
- **IDs**: UUID v4 as strings

---

## 2. URL Design

```
# Pattern: /api/{version}/{resource}/{id}/{sub-resource}

GET    /api/v1/proxy/orders              # list
GET    /api/v1/proxy/orders/:id          # get one
POST   /api/v1/proxy/orders              # create
PUT    /api/v1/proxy/orders/:id          # full update
PATCH  /api/v1/proxy/orders/:id          # partial update
DELETE /api/v1/proxy/orders/:id          # delete

# Actions (use nouns for REST resources, verbs for actions)
POST   /api/v1/proxy/orders/:id/cancel   # action
POST   /api/v1/proxy/orders/:id/renew    # action
POST   /api/v1/vps/instances/:id/reboot  # action
```

### Rules
- **Lowercase** with hyphens: `/proxy-orders` not `/proxyOrders`
- **Plural nouns** for collections: `/orders` not `/order`
- **No trailing slash**: `/orders` not `/orders/`
- **Max 3 levels deep**: `/resource/:id/sub-resource`

---

## 3. HTTP Status Codes

| Code | Usage |
|---|---|
| 200 OK | Successful GET, PUT, PATCH |
| 201 Created | Successful POST (sync) |
| 202 Accepted | Async operation started (provisioning) |
| 204 No Content | DELETE success |
| 400 Bad Request | Validation error |
| 401 Unauthorized | Missing/invalid auth |
| 403 Forbidden | Authenticated but not authorized |
| 404 Not Found | Resource not found |
| 409 Conflict | Duplicate (idempotency key exists) |
| 422 Unprocessable | Business rule violation |
| 429 Too Many Requests | Rate limit |
| 500 Internal Server Error | Unexpected error |
| 503 Service Unavailable | Upstream dependency down |

---

## 4. Response Format

### Success (single object)
```json
{
  "data": {
    "id": "uuid",
    "status": "active",
    "created_at": "2025-01-01T00:00:00Z"
  },
  "meta": {
    "request_id": "uuid"
  }
}
```

### Success (list)
```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "total_pages": 8
  },
  "meta": {
    "request_id": "uuid"
  }
}
```

### Async operation
```json
{
  "data": {
    "instance_id": "uuid",
    "job_id": "uuid",
    "status": "pending"
  },
  "meta": {
    "request_id": "uuid",
    "poll_url": "/api/v1/vps/instances/uuid/status"
  }
}
```

### Error
```json
{
  "error": {
    "code": "INSUFFICIENT_FUNDS",
    "message": "Wallet balance is insufficient for this order.",
    "details": {
      "required": 10.00,
      "available": 3.50
    },
    "request_id": "uuid"
  }
}
```

---

## 5. Error Codes

| Code | Status | Description |
|---|---|---|
| `VALIDATION_ERROR` | 400 | Input validation failed |
| `UNAUTHORIZED` | 401 | Auth required |
| `FORBIDDEN` | 403 | No permission |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Duplicate request |
| `INSUFFICIENT_FUNDS` | 422 | Wallet empty |
| `PROVIDER_UNAVAILABLE` | 503 | Proxy provider down |
| `NODE_UNAVAILABLE` | 503 | No Proxmox nodes |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Unexpected error |

---

## 6. Pagination

```
GET /api/v1/proxy/orders?page=1&limit=20&sort=created_at&order=desc

# Default: page=1, limit=20, max limit=100
# Sort fields must be whitelisted per endpoint
```

---

## 7. Versioning

- URL versioning: `/api/v1/`, `/api/v2/`
- Breaking changes require new version
- Old version maintained for 6 months after new release
- Deprecation header: `Deprecated: true` + `Sunset: 2026-01-01`

---

## 8. Required Headers

| Header | Required | Description |
|---|---|---|
| `Authorization: Bearer {token}` | Yes (auth endpoints) | JWT access token |
| `Content-Type: application/json` | Yes (POST/PUT/PATCH) | Request body type |
| `Idempotency-Key: {uuid}` | Recommended | Prevent duplicate orders |
| `X-Request-ID: {uuid}` | Auto-injected | Request tracing |
