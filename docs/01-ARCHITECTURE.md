# System Architecture — ProxyVPS Platform

**Document ID**: PVP-DOC-001  
**Version**: 1.0.0  
**Status**: Approved  
**Last Updated**: 2025-01-01  

---

## 1. Architecture Overview

ProxyVPS Platform sử dụng kiến trúc **Microservices** với message-driven communication qua NATS JetStream. Mỗi service có database riêng (database-per-service pattern), giao tiếp synchronous qua REST (internal) và asynchronous qua NATS events.

### Design Principles
1. **Single Responsibility**: Mỗi service chỉ làm một việc
2. **Event-Driven**: State changes publish events, không gọi trực tiếp
3. **Idempotency**: Mọi operation có thể retry an toàn
4. **Observability**: Every action produces a log entry
5. **Fail-Safe**: Lỗi provisioning phải hoàn tiền tự động (Saga pattern)

---

## 2. System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Client Layer                                  │
│   ┌─────────────┐  ┌─────────────────┐  ┌──────────────────────┐   │
│   │  User Web   │  │  Admin Dashboard │  │  Reseller Dashboard  │   │
│   │  (Next.js)  │  │   (Next.js)      │  │  / Reseller API      │   │
│   └──────┬──────┘  └────────┬─────────┘  └──────────┬───────────┘   │
└──────────┼──────────────────┼────────────────────────┼───────────────┘
           │                  │                        │
           │         HTTPS / WebSocket                 │
           ▼                  ▼                        ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      API Gateway :8080                               │
│   • JWT Validation     • Rate Limiting (per tier)                   │
│   • Request ID inject  • Route → service                            │
│   • TLS termination    • Access Log                                 │
└──────────────┬──────────────────────────────────────────────────────┘
               │  Internal HTTP (plain, within cluster)
     ┌─────────┴────────────────────────────────────────┐
     │                                                  │
     ▼                                                  ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐ ┌──────────────┐
│ User/Auth    │   │ Proxy        │   │ VPS          │ │ Reseller     │
│ Service      │   │ Service      │   │ Service      │ │ Service      │
│ :8081        │   │ :8082        │   │ :8083        │ │ :8084        │
└──────┬───────┘   └──────┬───────┘   └──────┬───────┘ └──────┬───────┘
       │                  │                  │                  │
       │           ┌──────┴───────┐   ┌──────┴──────┐         │
       │           │   Billing    │   │Notification  │         │
       │           │   Service    │   │   Service    │         │
       │           │   :8085      │   │   :8086      │         │
       │           └──────────────┘   └─────────────┘          │
       │                                                        │
       └───────────────────┬────────────────────────────────────┘
                           │
                  ┌────────▼────────┐
                  │   NATS JetStream │  ← Event Bus
                  │   :4222          │    All async events
                  └────────┬────────┘
                           │
                  ┌────────▼────────┐
                  │   Log Service   │  ← Consumes ALL events
                  │   :8087         │  → PostgreSQL + WebSocket
                  └─────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                    External Integrations                          │
│                                                                  │
│  ┌─────────────────────┐    ┌──────────────────────────────────┐ │
│  │  Proxy Providers    │    │   Proxmox Cluster                │ │
│  │  ┌──────────────┐   │    │   ┌────────┐  ┌────────┐        │ │
│  │  │ Provider A   │   │    │   │ Node 1 │  │ Node 2 │  ...   │ │
│  │  │ Provider B   │   │    │   └────────┘  └────────┘        │ │
│  │  │ Provider ... │   │    │   Proxmox VE 8.x REST API       │ │
│  │  └──────────────┘   │    └──────────────────────────────────┘ │
│  └─────────────────────┘                                         │
│                                                                  │
│  ┌─────────────────────┐    ┌──────────────────────────────────┐ │
│  │  Payment Gateways   │    │   Email (SMTP/SES)               │ │
│  │  • Stripe           │    │   Webhook delivery               │ │
│  │  • VNPay / MoMo     │    └──────────────────────────────────┘ │
│  └─────────────────────┘                                         │
└──────────────────────────────────────────────────────────────────┘
```

---

## 3. Service Inventory

| Service | Port | Responsibility | DB |
|---|---|---|---|
| api-gateway | 8080 | Routing, auth, rate limit | Redis (rate limit state) |
| user-service | 8081 | Users, auth, JWT, sessions | PostgreSQL |
| proxy-service | 8082 | Proxy orders, provider integration | PostgreSQL |
| vps-service | 8083 | VM lifecycle, Proxmox integration | PostgreSQL |
| billing-service | 8085 | Wallet, pricing, metering, invoices | PostgreSQL |
| log-service | 8087 | Event store, real-time streaming | PostgreSQL |
| notification-service | 8086 | Email, webhook, push alerts | PostgreSQL (queue) |
| reseller-service | 8084 | Reseller accounts, custom pricing | PostgreSQL |

---

## 4. Communication Patterns

### 4.1 Synchronous (HTTP REST)
Dùng cho: User-facing requests, admin queries, real-time reads

```
Client → API Gateway → Service → Response
```

### 4.2 Asynchronous (NATS JetStream)
Dùng cho: State changes, provisioning jobs, notifications

```
Service A → publish event → NATS → Service B consumes
```

**Tất cả events đều được Log Service consume** để tạo audit trail.

### 4.3 NATS Event Topics

| Topic | Publisher | Consumers |
|---|---|---|
| `order.proxy.created` | proxy-service | billing-service, log-service, notification-service |
| `order.proxy.fulfilled` | proxy-service | log-service, notification-service |
| `order.proxy.cancelled` | proxy-service | billing-service, log-service |
| `vm.provision.requested` | vps-service | vps-service (worker), log-service |
| `vm.state.changed` | vps-service | log-service, notification-service, billing-service |
| `payment.received` | billing-service | log-service, notification-service |
| `wallet.low_balance` | billing-service | notification-service |
| `user.registered` | user-service | log-service, notification-service |
| `reseller.created` | reseller-service | log-service, notification-service |

---

## 5. Request Flow Examples

### 5.1 User mua Proxy

```
1. POST /api/v1/proxy/orders
2. API Gateway: validate JWT, inject X-Request-ID
3. proxy-service: validate stock, check wallet via billing-service
4. billing-service: deduct wallet (optimistic lock)
5. proxy-service: call Provider Adapter → get credentials
6. proxy-service: store credentials, publish order.proxy.fulfilled
7. log-service: persist log entry
8. notification-service: send email
9. Response: 201 Created + credentials
```

### 5.2 User tạo VPS

```
1. POST /api/v1/vps/orders
2. API Gateway: validate JWT, inject X-Request-ID
3. vps-service: check resource availability on Proxmox nodes
4. billing-service: reserve funds (hold)
5. vps-service: publish vm.provision.requested → NATS queue
6. Response: 202 Accepted + job_id (async)

--- background job ---
7. vps-service worker: consume job, call Proxmox API
8. Poll VM status every 5s → emit vm.state.changed events
9. log-service: real-time push to user WebSocket
10. On VM RUNNING: confirm billing hold, inject SSH key
11. Publish vm.ready → notification-service sends email
```

---

## 6. Data Flow — Logging & Real-time

```
Any Service
    │ publish log.entry.{service}
    ▼
NATS JetStream
    │
    ▼
Log Service Consumer
    │
    ├──► PostgreSQL (persist, partitioned by month)
    │
    └──► WebSocket Hub
              │
              ├──► User connection: filter by user_id
              ├──► Admin connection: all events
              └──► Reseller connection: filter by reseller_id scope
```

---

## 7. Security Architecture

### 7.1 Authentication Flow
```
Login → user-service → JWT (access 15m + refresh 7d) → stored in httpOnly cookie
Request → API Gateway → validate JWT signature → inject user claims → forward
```

### 7.2 Authorization (RBAC)
| Role | Permissions |
|---|---|
| `super_admin` | All resources, all operations |
| `admin` | All resources except billing override |
| `reseller` | Own sub-users, own orders, own pricing |
| `user` | Own resources only |

### 7.3 API Key (Reseller)
- Reseller có thể generate API keys để integrate
- API key authenticate thay JWT
- Scope-limited: read-only hoặc full-access
- Rate limit riêng per API key

---

## 8. Deployment Architecture

```
Production:
┌─────────────────────────────────────────┐
│          Docker Compose                  │
│  (hoặc migrate K8s sau)                  │
│                                          │
│  • Nginx (reverse proxy + TLS)           │
│  • All microservices (Go binaries)       │
│  • PostgreSQL (primary + read replica)   │
│  • Redis (sentinel mode)                 │
│  • NATS (cluster mode, 3 nodes)         │
│  • Prometheus + Grafana                  │
└─────────────────────────────────────────┘
         │
         │  Private network
         ▼
┌─────────────────────────────────────────┐
│      Proxmox Cluster (10 nodes)          │
│      (self-hosted + dedicated servers)   │
└─────────────────────────────────────────┘
```

---

## 9. Technology Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Language | Go 1.22 | Performance, concurrency, team expertise |
| Message Bus | NATS JetStream | Lightweight, persistent, at-least-once delivery |
| Primary DB | PostgreSQL 16 | ACID compliance required for financial data |
| Cache | Redis 7 | Session, rate limit, hot data |
| Container | Docker | Simplicity for current scale |
| Frontend | Next.js 14 | SSR for SEO, TypeScript |
| Log Store | PostgreSQL (partitioned) | Avoid extra infrastructure complexity at this scale |

---

## 10. Quality Attributes

| Attribute | Target | Mechanism |
|---|---|---|
| Availability | 99.5% | Health checks, node failover, circuit breaker |
| Performance | p95 < 200ms | Redis cache, DB indexing, async provisioning |
| Scalability | Horizontal | Stateless services, external session store |
| Security | Zero plaintext | TLS everywhere, encrypted credentials at rest |
| Observability | Full trace | Request ID propagation, structured logging |
| Recoverability | RPO < 1h | PostgreSQL WAL, daily backups |
