# VPM Provider Adapter — Design Document

**Document ID**: PVP-DOC-022  
**Version**: 2.0.0  
**Component**: `proxy-service / providers / vpm`  
**Last Updated**: 2026-03-24

---

## 1. Overview

**VPS Proxy Manager (VPM)** là internal service quản lý proxy trực tiếp trên các node vật lý. Cloudmini tích hợp VPM dưới dạng một provider adapter (`vpm`), implement `IProxyProvider` interface.

- **Base URL**: `https://cz.resvn.net` (production)
- **Authentication**: Query-param `?access_code=vpm_...` (v2 dùng access_code thay Bearer)
- **API Docs**: `https://cz.resvn.net/billing-docs`
- **Provider type**: **Sync** — credentials trả về ngay sau `POST /api/v2/ipv4`

---

## 2. Adapter Structure

```
services/proxy-service/internal/providers/vpm/
├── types.go        # DTOs: CreateProxyV2Request, ProxySummaryV2, ProtocolConfig, APIError
├── client.go       # HTTP client: access_code auth, retry 3x, envelope unwrap, parseProxySummaries
├── adapter.go      # IProxyProvider impl: Purchase/Cancel/Suspend/Resume/CheckStatus
└── adapter_test.go # Unit tests
```

---

## 3. Interface Implementation

| Method | VPM API v2 Endpoint | Behavior |
|---|---|---|
| `Purchase` | `POST /api/v2/ipv4` hoặc `POST /api/v2/proxies` | Sync — credentials ngay lập tức |
| `Cancel` | `DELETE /api/v2/ipv4/{id}` | Permanent — releases IP/port |
| `Suspend` | `POST /api/v2/ipv4/{id}/suspend` | Temp pause (hoặc `PUT action=lock`) |
| `Resume` | `POST /api/v2/ipv4/{id}/resume` | Restore (hoặc `PUT action=unlock`) |
| `CheckStatus` | `GET /api/v2/ipv4/{id}` | Map status → Cloudmini constants |

---

## 4. VPM API v2 Endpoints

### 4.1 Create Proxy — Specific IP

```http
POST /api/v2/ipv4?access_code=<key>
Content-Type: application/json

{
  "ipv4": "103.190.120.56",           // required: specific IP
  "protocol": "default"               // "default"|"http"|"socks5"|"vmess"|"vless"|"shadowsocks"|"trojan"|"wireguard"
}
```

**Response** (single object):
```json
{
  "success": true,
  "data": {
    "id": "97b96318-f874-49ea-975d-e5e4d173bdca",
    "ipv4": "103.190.120.56",
    "username": "u_TroSFdkw",
    "password": "Sv4yAqOrSGbOedUH",
    "port_http": 47985,
    "port_socks": 46859,
    "protocol": "default",
    "connection_string": "http://u_TroSFdkw:Sv4yAqOrSGbOedUH@103.190.120.56:47985",
    "status": "completed"
  },
  "message": "default proxy created successfully"
}
```

> **Note:** `protocol=default` → single object với cả `port_http` + `port_socks`. Code parse bằng `parseProxySummaries()` → adapter tạo **2 credentials** (HTTP + SOCKS5).

### 4.2 Create Proxy — Region/Group Pool

```http
POST /api/v2/proxies?access_code=<key>
Content-Type: application/json

{
  "protocol": "default",
  "group_id": "bf10f37c-7a2e-4912-84c6-6d7ee6acc7d7"
}
```

**Response** (array):
```json
{
  "success": true,
  "data": [{
    "id": "...",
    "ipv4": "103.190.120.56",
    "port_http": 47213,
    "port_socks": 59220,
    "protocol": "default",
    ...
  }],
  "message": "1 proxy created successfully"
}
```

> **Note:** Response format có thể là array hoặc single object tuỳ endpoint. `parseProxySummaries()` tự detect `raw[0] == '['` để handle cả 2.

### 4.3 Get Proxy

```http
GET /api/v2/ipv4/{id}?access_code=<key>
GET /api/v2/ipv4/{ipv4}?access_code=<key>   # cũng hoạt động bằng IP trực tiếp
```

### 4.4 Delete Proxy

```http
DELETE /api/v2/ipv4/{id}?access_code=<key>
DELETE /api/v2/ipv4/{ipv4}?access_code=<key>  # IP trực tiếp
Response: 204 No Content
```

> 404 NOT_FOUND được treat là success (idempotent).

### 4.5 Lock / Unlock Proxy

```http
PUT /api/v2/ipv4/{ipv4}?access_code=<key>
Content-Type: application/json

{"action": "lock"}    # hoặc "unlock"
```

### 4.6 Stop / Start Proxy

```http
POST /api/v2/ipv4/{id}/stop?access_code=<key>
POST /api/v2/ipv4/{id}/start?access_code=<key>
```

### 4.7 List Groups/Regions

```http
GET /api/v1/groups?access_code=<key>
```

---

## 5. Protocol Behavior

| Protocol | port_http | port_socks | Credentials tạo ra |
|---|---|---|---|
| `default` | ✅ | ✅ | **2 credentials**: HTTP + SOCKS5 |
| `http` | ✅ | null | 1 credential: HTTP only |
| `socks5` | null | ✅ | 1 credential: SOCKS5 only |
| `vmess` / `vless` / `shadowsocks` / `trojan` / `wireguard` | — | — | 1 credential dùng `connection_string` |

---

## 6. Response Format Migration

VPM đang migrate response format:

| Endpoint | Old format | New format |
|---|---|---|
| `POST /api/v2/ipv4` | `data: [...]` array | `data: {...}` single object ✅ |
| `POST /api/v2/proxies` | `data: [...]` array | `data: [...]` array (giữ nguyên) |

Code dùng `parseProxySummaries(raw json.RawMessage)` detect tự động:
```go
if raw[0] == '[' {
    // parse array
} else {
    // parse single object → wrap thành []ProxySummaryV2{single}
}
```

---

## 7. PurchaseRequest Metadata Keys

| Key | Mô tả | Default |
|---|---|---|
| `ipv4` | Specific IP để tạo proxy | VPM auto-select từ pool |
| `group_id` | VPM region/group UUID | VPM auto-select |
| `protocol` | Protocol override: `"default"`, `"http"`, `"socks5"`, ... | `"default"` |
| `bandwidth_limit_mb` | Giới hạn bandwidth (MB), 0 = unlimited | 0 |
| `speed_limit_mbps` | Giới hạn tốc độ (Mbps) | 0 |

**Priority:**  
`metadata["protocol"]` > `product.Protocol` > `"default"`

---

## 8. Configuration

```env
VPM_BASE_URL=https://cz.resvn.net
VPM_API_KEY=vpm_xxxxxxxxxxxxx
```

```sql
INSERT INTO proxy.providers (id, name, display_name, adapter_type, config, is_active, priority)
VALUES (
    'b2000000-0000-0000-0000-000000000002',
    'vpm', 'VPS Proxy Manager', 'vpm',
    '{"base_url":"https://cz.resvn.net","api_key":""}',
    true, 10
);
```

> ⚠️ Registry key là **UUID** của provider trong DB — `order_usecase` lookup theo `product.ProviderID.String()`.

---

## 9. Error Handling

| HTTP Status / Code | Cloudmini Error | Ý nghĩa |
|---|---|---|
| 401, 403 | `ErrProviderUnavailable` | access_code sai |
| 400 `INVALID_INPUT` | `ErrInvalidConfig` | IP không available, protocol không hợp lệ |
| 404 `NOT_FOUND` | idempotent success (Cancel) / `ErrProviderUnavailable` (others) | Proxy không tồn tại |
| 5xx | retry 3x → `ErrProviderUnavailable` | VPM server lỗi |

---

## 10. Status Mapping

| VPM status | Cloudmini status |
|---|---|
| `completed` | `active` |
| `pending` | `processing` |
| `suspended` | `suspended` |
| `stopped` | `active` (port giữ nguyên) |
| `error` | `failed` |
| _(other)_ | `processing` |

---

## 11. Sync vs Async Comparison

| | VPM | Proxy-Cheap |
|---|---|---|
| **Credential delivery** | Sync (ngay lập tức) | Async (qua webhook) |
| **Order status after Purchase** | `active` | `processing` |
| **Cancel** | DELETE `/api/v2/ipv4/{id}` | No-op |
| **Webhook needed** | ❌ Không | ✅ Có |

---

## 12. UI Protocol Selector

Frontend (`OrderPanel`) hiển thị 3 nút protocol:
- **⚡ Default** — HTTP + SOCKS5 dual port (`protocol=default`)
- **🌐 HTTP** — HTTP only
- **🔌 SOCKS5** — SOCKS5 only

Mặc định là `default`. Protocol được gửi trong `metadata.protocol` → override `product.Protocol`.

---

## 13. Testing

```bash
# Build verification
go build ./...

# Manual: create proxy với specific IP
curl -s -X POST "https://cz.resvn.net/api/v2/ipv4?access_code=$VPM_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"ipv4": "103.190.120.56", "protocol": "default"}'

# Manual: lock/unlock by IP
curl -s -X PUT "https://cz.resvn.net/api/v2/ipv4/103.190.120.56?access_code=$VPM_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"action": "lock"}'

# Manual: delete by IP
curl -s -X DELETE "https://cz.resvn.net/api/v2/ipv4/103.190.120.56?access_code=$VPM_API_KEY"
```
