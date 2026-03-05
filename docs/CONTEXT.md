# CONTEXT.md — Current Development State

> **Living document** — cập nhật sau mỗi sprint hoặc milestone.  
> Dùng làm context bổ sung khi làm việc với AI tools.

**Last updated**: 2025-01-01  
**Current phase**: Phase 1 — Core Foundation  
**Sprint**: 1  

---

## ✅ What's Built (Production-Ready)
_Chưa có — dự án mới bắt đầu_

---

## 🚧 What's In Progress
- [ ] Project scaffolding (directory structure, go.mod per service)
- [ ] Shared packages: `pkg/nats`, `pkg/postgres`, `pkg/logger`, `pkg/middleware`
- [ ] Docker Compose setup (postgres, redis, nats)
- [ ] CI pipeline (GitHub Actions: lint, test, build)

---

## 📋 What's Not Built Yet
- User/Auth Service
- Billing Service (Wallet)
- Proxy Service + Provider Adapters
- VPS Service + Proxmox Adapter
- Log Service + WebSocket
- Notification Service
- Reseller Service
- API Gateway
- Frontend (Next.js)

---

## 📌 Recent Architecture Decisions

| Date | Decision | Rationale |
|---|---|---|
| 2025-01-01 | PostgreSQL schema-per-service (not DB-per-service) | Simpler ops at current scale, easier joins for admin queries |
| 2025-01-01 | NATS JetStream pull consumer | Better backpressure control vs push |
| 2025-01-01 | Prepaid wallet model | Eliminate chargeback risk, simpler billing logic |
| 2025-01-01 | slog over zerolog/zap | Stdlib, no extra dep, sufficient at this scale |

---

## ⚠️ Known Issues / Tech Debt
_None yet — project starting_

---

## 🔑 Key Environment Variables

```bash
# Common across services
DATABASE_URL=postgres://user:pass@localhost:5432/pvp
REDIS_URL=redis://localhost:6379
NATS_URL=nats://localhost:4222
JWT_SECRET=<secret>
ENCRYPTION_KEY=<32-byte-hex>  # for credential encryption

# Billing
STRIPE_SECRET_KEY=sk_...
VNPAY_TMN_CODE=...
VNPAY_HASH_SECRET=...

# Proxmox (per node)
PROXMOX_NODE_01_URL=https://192.168.1.10:8006
PROXMOX_NODE_01_TOKEN_ID=pvp@pam!pvp-token
PROXMOX_NODE_01_TOKEN_SECRET=<secret>
```

---

## 📂 Module Path

```
module github.com/org/proxy-vps-platform
```

---

## 🗓️ Update This File When:
- A service moves from "in progress" to "production-ready"
- A significant architecture decision is made
- A new tech debt item is identified
- Environment variables are added/changed
- A bug pattern is discovered (add to "lessons learned")
