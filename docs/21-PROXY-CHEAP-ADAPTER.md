# Proxy-Cheap Provider Adapter — Integration Design

**Document ID**: PVP-DOC-021  
**Version**: 1.1.0  
**Component**: `proxy-service/infrastructure/providers/proxy_cheap`  
**Status**: In Progress  
**Created**: 2026-03-12  
**Updated**: 2026-03-13  

---

## 1. Tổng Quan

Tài liệu này mô tả thiết kế tích hợp **Proxy-Cheap** (https://proxy-cheap.com) làm một provider adapter trong `proxy-service` của Cloudmini.

Proxy-Cheap cung cấp REST API tại `https://api.proxy-cheap.com` với xác thực qua header `X-Api-Key` + `X-Api-Secret`. Hệ thống hỗ trợ 6 loại proxy và push webhook khi có sự kiện thay đổi trạng thái.

### Các loại proxy hỗ trợ

> ⚠️ **Đơn vị thời gian: THÁNG**. Mọi order (static proxies) và gia hạn (`extend-period`) của Proxy-Cheap đều tính theo **tháng** (`periodInMonths`), không phải ngày. Rotating proxies tính theo GB traffic, không theo thời gian.

| Service ID | Nhãn | Mô hình tính tiền | Có Plan? |
|---|---|---|---|
| `static-residential-ipv4` | Static Residential (ISP) | per proxy × **tháng** | ✅ basic/standard/premium |
| `static-datacenter-ipv4` | Static Datacenter IPv4 | per proxy × **tháng** | ✅ basic/standard/premium |
| `static-datacenter-ipv6` | Static Datacenter IPv6 | per package (50/150/500) × **tháng** | ❌ (package-based) |
| `dedicated-mobile` | Static Mobile | per proxy × **tháng** | ✅ dedicated |
| `rotating-residential` | Rotating Residential | per **GB** traffic | ❌ (traffic-based) |
| `rotating-mobile` | Rotating Mobile | per **GB** traffic | ❌ (traffic-based) |

---

## 2. Authentication

```
Base URL: https://api.proxy-cheap.com
```

Mọi request (trừ một số endpoint public) cần 2 header:

```http
X-Api-Key:    <api_key>
X-Api-Secret: <api_secret>
```

Credentials lấy từ `https://app.proxy-cheap.com/api-keys` và được lưu encrypted trong `proxy.providers.config` (JSONB).

---

## 3. Toàn Bộ REST API Endpoints

### 3.1 Order Flow (v2 — Active)

> **Endpoint public** (không cần auth): `GET /v2/order` và `POST /v2/order/:serviceId/price`

#### `GET /v2/order` — Danh sách services & plans

```http
GET https://api.proxy-cheap.com/v2/order
Accept: application/json
```

Response:
```json
{
  "services": [
    { "id": "static-residential-ipv4", "label": "Static Residential (ISP)", "plans": [
        { "id": "basic",    "label": "Basic" },
        { "id": "standard", "label": "Dedicated" },
        { "id": "premium",  "label": "Premium" }
    ]},
    { "id": "rotating-mobile",    "label": "Rotating Mobile" },
    { "id": "rotating-residential","label": "Rotating Residential" },
    { "id": "static-datacenter-ipv4", "label": "Static Datacenter (IPv4)", "plans": [
        { "id": "basic" }, { "id": "standard" }, { "id": "premium" }
    ]},
    { "id": "static-datacenter-ipv6", "label": "Static Datacenter (IPv6)" },
    { "id": "dedicated-mobile", "label": "Static Mobile", "plans": [
        { "id": "dedicated", "label": "Dedicated" }
    ]}
  ]
}
```

---

#### `POST /v2/order/:serviceId` — Config options cho service

```http
POST https://api.proxy-cheap.com/v2/order/static-residential-ipv4
Content-Type: application/json

{ "planId": "basic" }
```

Response (static-residential-ipv4):
```json
{
  "serviceId": "static-residential-ipv4",
  "countries": ["US", "GB", "DE", "FR", "VN", "SG", ...],
  "isps": {
    "US": [
      { "id": "e649eed7-...", "label": "AT&T" },
      { "id": "e90710b4-...", "label": "Comcast" }
    ],
    "VN": [
      { "id": "23b0a5df-...", "label": "Zenlayer" }
    ]
  },
  "periods": {
    "months": [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12],
    "days": [7]
  }
}
```

Response (static-datacenter-ipv6):
```json
{
  "serviceId": "static-datacenter-ipv6",
  "packages": [
    { "id": "50",  "name": "50 proxies" },
    { "id": "150", "name": "150 proxies" },
    { "id": "500", "name": "500 proxies" }
  ],
  "countries": ["FR", "IT"],
  "periods": { "months": [1] }
}
```

Response (rotating-mobile / rotating-residential):
```json
{ "serviceId": "rotating-mobile" }
```
> Chỉ cần `traffic` (GB) khi đặt hàng.

---

#### `POST /v2/order/:serviceId/price` — Tính giá

```http
POST https://api.proxy-cheap.com/v2/order/static-residential-ipv4/price
Content-Type: application/json

{
  "planId":    "basic",
  "quantity":  1,
  "country":   "US",
  "period":    { "unit": "months", "value": 1 },
  "couponCode": ""
}
```

Response:
```json
{
  "finalPrice":              3.39,
  "priceNoDiscounts":        3.39,
  "discount":                0,
  "unitPrice":               3.39,
  "unitPriceAfterDiscount":  3.39,
  "additionalAmount":        0,
  "additionalAmountAfterDiscount": 0,
  "paymentFee":              0,
  "subtotal":                3.39,
  "discountAmount":          0,
  "finalPriceInCurrency":    3.39,
  "currency":                "USD",
  "appliedDiscounts": []
}
```

> ⚠️ **Lưu ý:** `paymentFee` có thể > 0 tùy phương thức thanh toán. Khi tính giá cho user trên Cloudmini, dùng `finalPrice` (đã bao gồm `paymentFee`).

Request cho rotating-mobile:
```json
{ "traffic": 3 }
```

Request cho static-datacenter-ipv6:
```json
{
  "packageId": "50",
  "country":   "FR",
  "period":    { "unit": "months", "value": 1 }
}
```

---

#### `POST /v2/order/:serviceId/execute` — Đặt hàng

```http
POST https://api.proxy-cheap.com/v2/order/static-residential-ipv4/execute
X-Api-Key:    <api_key>
X-Api-Secret: <api_secret>
Content-Type: application/json

{
  "planId":    "basic",
  "quantity":  1,
  "country":   "US",
  "ispId":     "e649eed7-ee2b-11ec-85c2-06902e647790",
  "period":    { "unit": "months", "value": 1 },
  "autoExtend": { "isEnabled": true },
  "traffic":   1,
  "couponCode": ""
}
```

Response:
```json
{
  "id":             "00000000-0000-0000-0000-000000000000",
  "periodInMonths": "1",
  "bandwidth":      "1",
  "totalPrice":     "3.39"
}
```

> ⚠️ Proxy được tạo **async**. Sau execute, proxy ở trạng thái `PENDING/INITIATING`. Cần webhook `proxy.status.changed` để biết khi nào `ACTIVE`.

---

### 3.2 Orders (read-only)

#### `GET /orders/:id` — Chi tiết order

```http
GET https://api.proxy-cheap.com/orders/{{orderId}}
X-Api-Key: <api_key>
X-Api-Secret: <api_secret>
```

Response:
```json
{
  "id":             "<order_uuid>",
  "periodInMoths":  "<number>",
  "bandwidth":      "<number>",
  "totalPrice":     "<float>"
}
```

#### `GET /orders/:id/proxies` — Proxies của order

Response:
```json
[
  {
    "id":          "<long>",
    "status":      "ACTIVE",
    "networkType": "RESIDENTIAL_STATIC",
    "authentication": {
      "whitelistedIps": [],
      "username": "user123",
      "password": "pass456"
    },
    "connection": {
      "publicIp":   "1.2.3.4",
      "connectIp":  "1.2.3.4",
      "ipVersion":  "IPv4",
      "httpPort":   10000,
      "httpsPort":  10001,
      "socks5Port": 10002
    },
    "proxyType": "HTTP",
    "createdAt": "2026-03-12T00:00:00Z",
    "expiresAt": "2026-04-12T00:00:00Z",
    "metadata": { "ispName": "AT&T" }
  }
]
```

---

### 3.3 Proxy Management

#### `GET /proxies` — Tất cả proxies active

#### `GET /proxies/:id` — Chi tiết proxy (có thêm bandwidth info)

```json
{
  "id":          "<long>",
  "status":      "ACTIVE",
  "networkType": "RESIDENTIAL_STATIC",
  "countryCode": "US",
  "authentication": { ... },
  "connection": { ... },
  "proxyType": "HTTP",
  "createdAt": "...",
  "expiresAt": "...",
  "metadata": { "ispName": "..." },
  "bandwidth": {
    "total": 10,
    "used":  3
  }
}
```

**Proxy statuses:** `PENDING` → `INITIATING` → `ACTIVE` → `EXPIRED` / `CANCELED`

#### `POST /proxies/:id/extend-period`
```json
{ "periodInMonths": 1, "couponCode": "" }
```

#### `POST /proxies/:id/period-extension-price`
```json
{ "periodInMonths": 1 }
```
Response: `{ "finalPrice": <n>, "priceNoDiscounts": <n>, "discount": <n> }`

#### `POST /proxies/:id/buy-bandwidth`
```json
{ "bandwidth": 5 }
```

#### `POST /proxies/:id/bandwidth-price`
```json
{ "bandwidth": 5 }
```
Response: `{ "finalPrice": <n>, "priceNoDiscounts": <n>, "discount": <n> }`

#### `POST /proxies/:id/whitelist-ip`
```json
{ "ips": ["1.2.3.4", "5.6.7.8"] }
```

#### `GET /proxies/:id/change-protocol`
Response: `{ "currentType": "HTTP", "availableTypes": ["HTTP", "SOCKS5"] }`

#### `POST /proxies/:id/change-protocol`
```json
{ "newType": "SOCKS5" }
```

#### `GET /proxies/:id/change-authentication-type`
Response:
```json
{
  "currentAuthenticationType": "USERNAME_PASSWORD",
  "availableAuthenticationTypes": ["IP_WHITELIST", "USERNAME_PASSWORD"]
}
```

#### `POST /proxies/:id/change-authentication-type`
```json
{ "newAuthType": "IP_WHITELIST", "ips": ["1.2.3.4"] }
```

#### `POST /proxies/:id/auto-extend/enable` — `204 No Content`

#### `POST /proxies/:id/auto-extend/disable` — `204 No Content`

#### `POST /proxies/:id/rotate-ip`
> Chỉ cho **dedicated mobile**. Assigns new public IP. `204 No Content`.

---

### 3.4 Account

#### `GET /account/balance`

```http
GET https://api.proxy-cheap.com/account/balance
X-Api-Key: <api_key>
```

Response: `{ "balance": "<integer>" }` (USD cents)

---

## 4. Webhooks

Proxy-Cheap gửi webhook đến endpoint của Cloudmini (cấu hình tại `https://app.proxy-cheap.com/webhooks`).

### 4.1 Cấu Trúc Request

```http
POST <cloudmini_webhook_url>
Webhook-Event:     proxy.status.changed
Webhook-Id:        <event_uuid>
Webhook-Signature: sha256=<hmac>
Content-Type:      application/json
```

### 4.2 Xác Thực HMAC

```
HMAC input = algo + event_name + event_id + body + secret

Ví dụ:
input = "sha256" + "proxy.status.changed" + "<event_id>" + "<minified_json_body>" + "<webhook_secret>"
signature = "sha256=" + HMAC_SHA256(input, webhook_secret)
```

> ⚠️ Body phải là **minified JSON** (không pretty-print) khi tính HMAC.
> Test secret: `0qFbBP1zE5MuSMy`

### 4.3 Events

#### `proxy.status.changed`
```json
{
  "proxyId":   1410379,
  "oldStatus": "PENDING",
  "status":    "ACTIVE"
}
```
Statuses: `PENDING` → `INITIATING` → `ACTIVE` → `EXPIRED` / `CANCELED`

#### `proxy.bandwidth.added`
```json
{
  "proxyId":    1410379,
  "trafficInGb": 1
}
```
Triggered khi auto-extend thêm bandwidth.

#### `proxy.maintenance_window.created`
```json
{
  "proxyId":             1410379,
  "maintenanceWindowId": "01985631-...",
  "startsAt":            "2025-07-29T00:00:00+00:00",
  "endsAt":              "2025-07-31T00:00:00+00:00"
}
```
Server down → default window 24H.

#### `proxy.maintenance_window.cancelled`
```json
{
  "proxyId":             1410379,
  "maintenanceWindowId": "01985631-..."
}
```
Server back online.

> ⚠️ Proxy-Cheap **không retry webhook**. Cloudmini PHẢI implement polling fallback.

---

## 5. Thiết Kế Adapter

### 5.1 File Structure

```
services/proxy-service/
├── internal/
│   ├── infrastructure/
│   │   └── providers/
│   │       └── proxy_cheap/
│   │           ├── adapter.go       ← IProxyProvider implementation
│   │           ├── client.go        ← HTTP client (retry, auth)
│   │           ├── catalog.go       ← Service/product catalog sync
│   │           ├── webhook.go       ← Webhook handler + HMAC verify
│   │           ├── mapper.go        ← Map PC types ↔ Cloudmini types
│   │           └── adapter_test.go  ← Unit tests với mock HTTP
│   └── handler/http/
│       └── webhook_handler.go       ← POST /api/v1/proxy/webhooks/proxy-cheap
```

### 5.2 Adapter Interface Implementation

```go
// adapter.go
type ProxyCheapAdapter struct {
    client        *ProxyCheapClient
    catalogCache  *CatalogCache
    webhookSecret string
}

func (a *ProxyCheapAdapter) Name() string { return "proxy_cheap" }

// Purchase → POST /v2/order/:serviceId/execute
func (a *ProxyCheapAdapter) Purchase(ctx context.Context, req PurchaseRequest) (*PurchaseResult, error) {
    // 1. Map PurchaseRequest → ProxyCheapOrderRequest
    // 2. Call /v2/order/:serviceId/price để verify totalPrice
    // 3. Call /v2/order/:serviceId/execute
    // 4. Return PurchaseResult{ProviderOrderID: resp.ID, Status: "pending"}
    // NOTE: Credentials chưa có ngay — cần đợi webhook proxy.status.changed → ACTIVE
}

// GetCredentials → GET /orders/:id/proxies
func (a *ProxyCheapAdapter) GetCredentials(ctx context.Context, providerOrderID string) (*Credentials, error) {
    proxies, err := a.client.GetOrderProxies(ctx, providerOrderID)
    // Map connection info → Credentials{Host, Port, Username, Password, Protocol}
}

// CheckOrderStatus → GET /orders/:id/proxies → lấy status từ proxy đầu tiên
func (a *ProxyCheapAdapter) CheckOrderStatus(ctx context.Context, providerOrderID string) (*OrderStatus, error)

// Extend → POST /proxies/:proxyId/extend-period
func (a *ProxyCheapAdapter) Extend(ctx context.Context, providerOrderID string, req ExtendRequest) error

// GetBalance → GET /account/balance
func (a *ProxyCheapAdapter) GetBalance(ctx context.Context) (*ProviderBalance, error)

// GetProducts → GET /v2/order (+ POST /v2/order/:serviceId cho từng service)
func (a *ProxyCheapAdapter) GetProducts(ctx context.Context) ([]ProviderProduct, error)
```

### 5.3 Provider Config (lưu encrypted trong DB)

```json
{
  "api_key":        "encrypted:...",
  "api_secret":     "encrypted:...",
  "webhook_secret": "encrypted:...",
  "base_url":       "https://api.proxy-cheap.com",
  "timeout_seconds": 30,
  "retry_count":    3
}
```

### 5.4 Type Mapping

| Proxy-Cheap | Cloudmini |
|---|---|
| `RESIDENTIAL_STATIC` | `residential / static` |
| `RESIDENTIAL` (rotating) | `residential / rotating` |
| `DATACENTER` (IPv4) | `datacenter / ipv4` |
| `DATACENTER` (IPv6) | `datacenter / ipv6` |
| `MOBILE` (static) | `mobile / static` |
| `MOBILE` (rotating) | `mobile / rotating` |
| `HTTP` | `http` |
| `SOCKS5` | `socks5` |
| `PENDING/INITIATING` | `provisioning` |
| `ACTIVE` | `fulfilled` |
| `EXPIRED` | `expired` |
| `CANCELED` | `cancelled` |

### 5.5 Product Catalog Sync

```go
// Chạy khi startup + mỗi 6 giờ
func (c *CatalogCache) Refresh(ctx context.Context) error {
    // 1. GET /v2/order → lấy danh sách services + plans
    // 2. Với mỗi service + plan: POST /v2/order/:serviceId → lấy countries, ISPs, packages
    // 3. Với mỗi combination: POST /v2/order/:serviceId/price → lấy base_cost
    // 4. Upsert vào proxy.products table
}
```

---

## 6. Webhook Handler

### 6.1 Endpoint

```
POST /api/v1/proxy/webhooks/proxy-cheap
```

Endpoint public (không cần JWT), verify bằng HMAC.

### 6.2 Flow xử lý

```
Nhận webhook
    │
    ▼
Verify HMAC signature
    │ invalid → 401
    ▼
Parse Webhook-Event header
    │
    ├── proxy.status.changed
    │       │
    │       ▼ status == ACTIVE
    │   GET /orders/:orderId/proxies → lấy credentials
    │   Encrypt credentials → update Order.credentials
    │   Update Order.status = fulfilled
    │   Publish NATS: proxy.order_fulfilled
    │       │
    │       ▼ status == EXPIRED/CANCELED
    │   Update Order.status = expired/cancelled
    │   Publish NATS: proxy.order_expired
    │
    ├── proxy.bandwidth.added
    │   Update Order metadata (bandwidth info)
    │
    ├── proxy.maintenance_window.created
    │   Log + optionally notify user
    │
    └── proxy.maintenance_window.cancelled
        Log
```

### 6.3 HMAC Verification Code

```go
func VerifyWebhookSignature(r *http.Request, body []byte, secret string) bool {
    sig := r.Header.Get("Webhook-Signature")
    eventName := r.Header.Get("Webhook-Event")
    eventID := r.Header.Get("Webhook-Id")

    // Input: algo + eventName + eventID + body + secret
    // "sha256" là literal string, không phải func name
    input := "sha256" + eventName + eventID + string(body) + secret

    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write([]byte(input))
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

    return hmac.Equal([]byte(sig), []byte(expected))
}
```

---

## 7. Polling Fallback

Vì Proxy-Cheap **không retry webhook**, cần polling fallback:

```
Job: sync-proxy-cheap-pending-orders
Schedule: Mỗi 5 phút
Logic:
  - Tìm Orders có provider=proxy_cheap, status=provisioning, created_at > 30 phút trước
  - Với mỗi order: GET /orders/:id/proxies
  - Nếu proxy.status == ACTIVE → xử lý như webhook proxy.status.changed
  - Nếu order > 24 giờ mà vẫn PENDING → mark failed + refund
```

---

## 8. Billing Model — Monthly Cycle

### 8.1 Order (Static Proxies)

Proxy-Cheap tính tiền theo **tháng**. Khi đặt hàng:

```json
POST /v2/order/static-residential-ipv4/execute
{
  "planId":     "basic",
  "quantity":   1,
  "country":    "US",
  "ispId":      "<uuid-optional>",
  "period":     { "unit": "months", "value": 1 },
  "autoExtend": { "isEnabled": true }
}
```

Response trả về `periodInMonths` — proxy active trong N tháng kể từ khi `ACTIVE`.

### 8.2 Gia hạn (Renew)

```json
POST /proxies/:proxyId/extend-period
{ "periodInMonths": 1, "couponCode": "" }
```

> Endpoint Cloudmini: `POST /api/v1/proxy/orders/{id}/renew`  
> Billing: deduct wallet trước khi gọi proxy-cheap extend.

### 8.3 Country & ISP Selection

> **Quan trọng:** Country và ISP KHÔNG được định nghĩa ở product config.
> Admin chỉ định nghĩa **service + plan**. Client chọn **country và ISP** khi đặt hàng.

Flow lấy options cho client:
```
GET /api/v1/proxy/service-options?service_id=static-residential-ipv4&plan_id=basic
→ proxy-service → POST /v2/order/:serviceId (proxy-cheap API)
→ Response: { countries: ["US","VN",...], isps: { "US": [{id, label},...] } }
```

Khi ISP = none (client không chọn ISP cụ thể) → `ispId` không gửi trong execute request → proxy-cheap assign ISP ngẫu nhiên theo country.

---

## 9. Order Creation Flow (Tích Hợp Billing)

```
User → POST /api/v1/proxy/orders
    │  Body: { product_id, quantity, metadata: { country, isp_id?, period_months } }
    ▼
proxy-service:
  1. Validate product (proxy.products) → lấy metadata: service_id, plan_id
  2. Merge product.metadata + order.metadata (country, isp_id, period_months)
  3. Tính giá: POST /v2/order/:serviceId/price (verify với period_months)
  4. Apply markup → sell_price (billing-service tính)
  5. billing-service: deduct wallet (hold)
  6. POST /v2/order/:serviceId/execute → {id, periodInMonths, totalPrice}
  7. Lưu Order {status=provisioning, provider_order_id=id, expires_at = now + periodInMonths}
  8. Trả response cho user (status=provisioning)

[Async - khi webhook đến]
  9. proxy.status.changed (ACTIVE) → GET /orders/:id/proxies
  10. Encrypt credentials → lưu DB
  11. Order.status = fulfilled
  12. billing-service: confirm charge (release hold)
  13. NATS: proxy.order_fulfilled → notification-service → email user

[Fallback nếu execute fail]
  billing-service: refund (cancel hold)
  Order.status = failed
```

---

## 9. Cấu Hình & Environment Variables

```env
# Proxy-Cheap credentials (lưu encrypted trong DB, không hardcode env)
# Admin nhập qua /admin/proxy/providers UI
# Stored in proxy.providers.config JSONB (encrypted AES-256-GCM)

# Webhook endpoint (đăng ký tại app.proxy-cheap.com/webhooks)
PROXY_CHEAP_WEBHOOK_URL=https://<domain>/api/v1/proxy/webhooks/proxy-cheap
```

---

## 10. Error Handling

| Proxy-Cheap Error | HTTP Code | Cloudmini Error |
|---|---|---|
| Invalid credentials | 403 | `ErrProviderUnavailable` |
| Validation error | 422 | `ErrInvalidLocation` / `ErrInvalidConfig` |
| Insufficient balance (PC account) | 402 | `ErrProviderBalance` → Alert admin |
| Rate limit | 429 | Retry sau exponential backoff |
| Server error | 5xx | `ErrProviderUnavailable` → Fallback provider |

```go
func (c *ProxyCheapClient) handleError(resp *http.Response) error {
    switch resp.StatusCode {
    case 403: return ErrProviderUnavailable
    case 422: return parseValidationError(resp.Body)
    case 429: return ErrRateLimited
    default:  return ErrProviderUnavailable
    }
}
```

---

## 11. Testing

```bash
# Unit tests với mock HTTP server
go test ./services/proxy-service/internal/infrastructure/providers/proxy_cheap/...

# Integration test (cần credentials thật)
PROXY_CHEAP_API_KEY=... PROXY_CHEAP_API_SECRET=... \
go test -tags=integration ./services/proxy-service/...

# Test webhook signature verification
go test -run TestVerifyWebhookSignature ./...
```

### Test Cases

- [ ] `Purchase` → success response mapping
- [ ] `Purchase` → 422 validation error handling
- [ ] `GetCredentials` → map proxy connection info
- [ ] `CheckOrderStatus` → PENDING, ACTIVE, EXPIRED mapping
- [ ] `Extend` → extend period call
- [ ] `GetBalance` → account balance
- [ ] `GetProducts` → catalog sync (all 6 services)
- [ ] Webhook HMAC verify → valid signature
- [ ] Webhook HMAC verify → invalid signature → reject
- [ ] Webhook handler → `proxy.status.changed` ACTIVE → update order
- [ ] Polling fallback → pending order > 30 min → poll status

---

## 12. Checklist Triển Khai

- [ ] Implement `ProxyCheapClient` (HTTP client + auth + retry)
- [ ] Implement `ProxyCheapAdapter` (tất cả methods của `IProxyProvider`)
- [ ] Implement `CatalogCache` (sync products mỗi 6h)
- [ ] Implement `WebhookHandler` (HMAC verify + event routing)
- [ ] Add polling fallback cron job
- [ ] Register adapter trong `ProviderRegistry`
- [ ] Add Provider row vào DB (`proxy.providers`)
- [ ] Populate `proxy.products` từ catalog sync
- [ ] Đăng ký webhook URL tại app.proxy-cheap.com/webhooks
- [ ] Unit tests (coverage ≥ 80%)
- [ ] Integration test với sandbox/live credentials
- [ ] Update CHANGELOG.md

---

## 13. Tham Khảo

- **Postman Collection**: `/opt/Cloudmini/Proxy-Cheap API.postman_collection.json`
- **API Docs**: https://docs.proxy-cheap.com/
- **Provider Adapters Design**: `docs/12-PROVIDER-ADAPTERS.md`
- **Proxy Service Design**: `docs/05-PROXY-SERVICE.md`
- **API Key Management**: https://app.proxy-cheap.com/api-keys
- **Webhook Config**: https://app.proxy-cheap.com/webhooks
