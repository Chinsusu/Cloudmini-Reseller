# Log Service — Service Design

**Document ID**: PVP-DOC-008  
**Version**: 1.0.0  
**Service**: log-service  
**Port**: 8087  

---

## 1. Responsibilities

- Consume tất cả events từ NATS và persist vào PostgreSQL
- Real-time WebSocket streaming tới dashboard
- Query API cho audit logs (search, filter, paginate)
- Role-based data visibility (user chỉ thấy log của mình)
- Structured log format chuẩn toàn hệ thống

---

## 2. Architecture

```
All Services
    │ publish to NATS topics
    ▼
NATS JetStream Consumer (durable, at-least-once)
    │
    ├──► PostgreSQL  (persist, partition by month)
    │
    └──► WebSocket Hub
              │
              ├── user connection    → filter: user_id = X
              ├── admin connection   → all events
              └── reseller connection → filter: reseller_id = X
```

---

## 3. Log Entry Structure

```json
{
  "id": "uuid",
  "request_id": "uuid",
  "trace_id": "uuid",
  "service_name": "proxy-service",
  "user_id": "uuid",
  "reseller_id": "uuid | null",
  "actor_type": "user",
  "actor_id": "uuid",
  "action": "order.proxy.created",
  "level": "INFO",
  "resource_type": "proxy_order",
  "resource_id": "uuid",
  "message": "Proxy order created successfully",
  "payload": {
    "product_id": "uuid",
    "provider": "smartproxy",
    "amount": 5.00
  },
  "duration_ms": 145,
  "ip_address": "1.2.3.4",
  "created_at": "2025-01-01T00:00:00Z"
}
```

---

## 4. API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/v1/logs` | User's own logs |
| GET | `/api/v1/logs/:id` | Log entry detail |
| GET | `/api/v1/logs/orders/:order_id` | Logs for specific order |
| GET | `/api/v1/admin/logs` | All logs (admin only) |
| GET | `/ws/events` | WebSocket real-time stream |

### Query Parameters (GET /logs)
```
?action=order.proxy.created
&level=ERROR
&resource_type=proxy_order
&resource_id=uuid
&from=2025-01-01T00:00:00Z
&to=2025-01-31T23:59:59Z
&page=1
&limit=50
```

---

## 5. WebSocket Protocol

### Connection
```
GET /ws/events
Headers:
  Authorization: Bearer {jwt}
  or
  ?token={jwt}  (for browser EventSource fallback)
```

### Message Format (server → client)
```json
{
  "type": "log_entry",
  "data": {
    "action": "vm.state.changed",
    "level": "INFO",
    "message": "VPS is now booting",
    "resource_id": "instance-uuid",
    "created_at": "2025-01-01T00:00:01Z"
  }
}
```

### Client Filter (client → server)
```json
{
  "type": "subscribe",
  "filter": {
    "resource_id": "instance-uuid",
    "resource_type": "vps_instance"
  }
}
```

---

## 6. Standard Action Names

Tất cả services phải dùng action names theo format: `{resource}.{event}`

| Action | Service |
|---|---|
| `user.registered` | user-service |
| `user.login` | user-service |
| `user.suspended` | user-service |
| `order.proxy.created` | proxy-service |
| `order.proxy.fulfilled` | proxy-service |
| `order.proxy.failed` | proxy-service |
| `order.proxy.cancelled` | proxy-service |
| `vm.provision.requested` | vps-service |
| `vm.state.changed` | vps-service |
| `vm.ready` | vps-service |
| `vm.terminated` | vps-service |
| `billing.charged` | billing-service |
| `billing.deposit.completed` | billing-service |
| `billing.wallet.low` | billing-service |
| `billing.wallet.empty` | billing-service |
| `reseller.created` | reseller-service |
| `reseller.pricing.updated` | reseller-service |

---

## 7. Retention & Archival

- **Hot data**: 3 months (fast query, current partition)
- **Warm data**: 3-12 months (older partitions, slightly slower)
- **Cold data**: > 12 months (archive to S3/object storage as JSONL)
- Partitions dropped after archival to free disk space
