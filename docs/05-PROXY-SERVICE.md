# Proxy Service — Service Design

**Document ID**: PVP-DOC-005  
**Version**: 1.0.0  
**Service**: proxy-service  
**Port**: 8082  

---

## 1. Responsibilities

- Quản lý proxy product catalog
- Nhận và xử lý proxy orders
- Routing order đến provider phù hợp
- Provider Adapter pattern — abstraction layer cho nhiều providers
- Credential delivery và lưu trữ encrypted
- Order lifecycle: active → expired/cancelled
- Expiry monitoring và renewal reminders

---

## 2. API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/v1/proxy/products` | Danh sách sản phẩm proxy |
| GET | `/api/v1/proxy/products/:id` | Chi tiết sản phẩm |
| POST | `/api/v1/proxy/orders` | Tạo order mới |
| GET | `/api/v1/proxy/orders` | Danh sách orders của user |
| GET | `/api/v1/proxy/orders/:id` | Chi tiết order + credentials |
| POST | `/api/v1/proxy/orders/:id/renew` | Gia hạn order |
| POST | `/api/v1/proxy/orders/:id/cancel` | Hủy order |
| GET | `/api/v1/admin/proxy/orders` | Admin: tất cả orders |
| GET | `/api/v1/admin/proxy/providers` | Quản lý providers |

---

## 3. Provider Abstraction Layer

```go
// IProxyProvider — mọi provider đều implement interface này
type IProxyProvider interface {
    Purchase(ctx context.Context, req PurchaseRequest) (*PurchaseResult, error)
    GetCredentials(ctx context.Context, providerOrderID string) (*Credentials, error)
    CheckStatus(ctx context.Context, providerOrderID string) (*OrderStatus, error)
    Extend(ctx context.Context, providerOrderID string, days int) error
    Cancel(ctx context.Context, providerOrderID string) error
    GetBalance(ctx context.Context) (*ProviderBalance, error)
}
```

### Provider Router Logic
```
Order request
    │
    ▼
Filter providers by: proxy_type, protocol, location
    │
    ▼
Rank by: priority > availability > cost
    │
    ▼
Select provider → call adapter
    │ (fallback if provider fails)
    ▼
Next available provider
```

---

## 4. Order Lifecycle

```
                  ┌─────────┐
                  │ pending │  ← order created, payment not confirmed
                  └────┬────┘
                       │ billing confirms deduction
                  ┌────▼──────────┐
                  │  processing   │  ← calling provider API
                  └────┬──────────┘
            ┌──────────┴──────────┐
            │ success             │ failure
       ┌────▼────┐          ┌─────▼───────┐
       │  active │          │   failed    │  → auto refund
       └────┬────┘          └─────────────┘
            │ expires
       ┌────▼────────┐
       │   expired   │
       └─────────────┘
            │ user cancels (if active)
       ┌────▼────────┐
       │  cancelled  │
       └─────────────┘
```

---

## 5. Order Creation Flow (Saga)

```
Step 1: Validate product exists + is available
Step 2: Calculate price (apply reseller pricing if applicable)
Step 3: Call billing-service → deduct wallet
        (compensating action: refund if steps 4-5 fail)
Step 4: Call Provider Adapter → purchase
Step 5: Store credentials (encrypted with AES-256)
Step 6: Update order status = active
Step 7: Publish order.proxy.fulfilled
```

### Idempotency
- Client gửi `Idempotency-Key` header (UUID)
- Service check xem key đã tồn tại chưa → trả response cũ nếu có
- Ngăn double-order khi client retry

---

## 6. Credential Storage

Credentials (host, port, username, password) được encrypt bằng AES-256-GCM trước khi lưu vào PostgreSQL JSONB column. Encryption key quản lý qua environment variable hoặc Vault.

```json
// Stored (encrypted)
{
  "encrypted": "base64encodeddata...",
  "iv": "base64iv..."
}

// Returned to user (decrypted)
{
  "host": "proxy.provider.com",
  "port": 10000,
  "username": "user123",
  "password": "pass456",
  "protocol": "socks5"
}
```

---

## 7. Expiry Monitoring (Cron Jobs)

| Job | Schedule | Action |
|---|---|---|
| `check-expiring-proxies` | Every hour | Find orders expiring in 3 days → publish reminder |
| `expire-overdue-orders` | Every 30 min | Mark expired orders as `expired` |
| `sync-provider-status` | Every 6 hours | Poll provider for status sync |

---

## 8. Events Published

| Event | Trigger |
|---|---|
| `order.proxy.created` | Order created |
| `order.proxy.fulfilled` | Credentials delivered |
| `order.proxy.failed` | Provider error |
| `order.proxy.expired` | Order expired |
| `order.proxy.cancelled` | User cancelled |
| `order.proxy.expiry_reminder` | 3 days before expiry |
