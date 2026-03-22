# VPM Provider Adapter — Design Document

**Document ID**: PVP-DOC-022  
**Version**: 1.0.0  
**Component**: proxy-service / providers / vpm  

---

## 1. Overview

**VPS Proxy Manager (VPM)** là internal service quản lý proxy trực tiếp trên các node vật lý. Cloudmini tích hợp VPM dưới dạng một provider adapter (`vpm`), implement `IProxyProvider` interface.

- **Base URL**: `https://cz.resvn.net` (production)
- **Authentication**: Bearer Token (`Authorization: Bearer vpm_...`)
- **Provider type**: **Sync** — credentials trả về ngay sau `POST /proxies`

---

## 2. Adapter Structure

```
services/proxy-service/internal/providers/vpm/
├── types.go       # DTOs: CreateProxyRequest, ProxySummary, APIError
├── client.go      # HTTP client: Bearer auth, retry 3x, envelope unwrap
├── adapter.go     # IProxyProvider impl: Purchase/Cancel/CheckStatus
└── adapter_test.go # 13 unit tests
```

---

## 3. Interface Implementation

| Method | VPM API | Behavior |
|---|---|---|
| `Purchase` | `POST /api/v1/proxies` | Sync — credentials returned immediately |
| `Cancel` | `DELETE /api/v1/proxies/{id}` | Permanent — releases IP/port |
| `CheckStatus` | `GET /api/v1/proxies/{id}` | Map status → Cloudmini constants |

---

## 4. VPM API Endpoints Used

### Create Proxy
```
POST /api/v1/proxies
Authorization: Bearer <api_key>

{
  "protocol": "socks5" | "http",   // required
  "ip_range_id": "<uuid>",          // optional: VPM auto-selects if empty
  "node_id": "<uuid>",              // optional
  "auth_user": "string",            // optional: auto-generated if empty
  "auth_pass": "string",            // optional: auto-generated if empty
  "bandwidth_limit_mb": 0,          // optional: 0 = unlimited
  "speed_limit_mbps": 0             // optional
}

Response 201:
{
  "success": true,
  "data": {
    "id": "<proxy-uuid>",
    "host": "103.228.75.202",
    "port": 22050,
    "protocol": "socks5",
    "auth_user": "user_abc123",
    "auth_pass": "secret-password",
    "outbound_ip": "...",
    "status": "running"
  }
}
```

### Delete Proxy
```
DELETE /api/v1/proxies/{id}
Response 204: No Content
```

### Get Proxy Status
```
GET /api/v1/proxies/{id}
Response 200: same as create (without auth_pass)
```

### Stop / Start Proxy (optional — for suspend/resume)
```
POST /api/v1/proxies/{id}/stop
POST /api/v1/proxies/{id}/start
Response 200
```

---

## 5. Status Mapping

| VPM status | Cloudmini status |
|---|---|
| `running` | `active` |
| `stopped` | `active` (proxy reserved, port kept) |
| `creating` | `processing` |
| `error` | `failed` |
| _(other)_ | `processing` |

---

## 6. PurchaseRequest Metadata Keys

Khi tạo order với VPM product, frontend/admin có thể pass metadata:

| Key | Mô tả | Default |
|---|---|---|
| `ip_range_id` | VPM IP range UUID | VPM auto-select |
| `node_id` | Target node UUID | VPM auto-select |
| `bandwidth_limit_mb` | Giới hạn bandwidth (MB), 0 = unlimited | 0 |
| `speed_limit_mbps` | Giới hạn tốc độ (Mbps) | 0 |
| `auth_user` | Username tùy chọn | VPM auto-generate |
| `auth_pass` | Password tùy chọn | VPM auto-generate |

---

## 7. Configuration

### Environment Variables
```env
VPM_BASE_URL=https://cz.resvn.net
VPM_API_KEY=vpm_xxxxxxxxxxxxx
```

### Database (proxy.providers)
```sql
INSERT INTO proxy.providers (id, name, display_name, adapter_type, config, is_active, priority)
VALUES (
    'b2000000-0000-0000-0000-000000000002',
    'vpm',
    'VPS Proxy Manager',
    'vpm',
    '{"base_url":"https://cz.resvn.net","api_key":""}',
    true,
    10
);
```

> ⚠️ Registry key phải là **UUID** của provider trong DB (không phải string `"vpm"`), vì `order_usecase.CreateOrder` lookup theo `product.ProviderID.String()`.

---

## 8. Error Handling

| HTTP Status | Cloudmini Error | Ý nghĩa |
|---|---|---|
| 401, 403 | `ErrProviderUnavailable` | API key sai hoặc hết hạn |
| 400, 422 | `ErrInvalidConfig` | Request không hợp lệ (sai protocol, range không tồn tại) |
| 404 | `ErrProviderUnavailable` | Proxy không tồn tại |
| 5xx | retry 3x → `ErrProviderUnavailable` | VPM server lỗi |

---

## 9. Sync vs Async Comparison

| | VPM | Proxy-Cheap |
|---|---|---|
| **Credential delivery** | Sync (ngay lập tức) | Async (qua webhook) |
| **Order status after Purchase** | `active` | `processing` |
| **Cancel** | DELETE proxy | No-op (no cancel API) |
| **Webhook needed** | ❌ Không | ✅ Có |

---

## 10. Testing

```bash
# Unit tests (không cần server thật)
go test ./internal/providers/vpm/... -v

# Manual: kiểm tra API key hợp lệ
curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer $VPM_API_KEY" \
  "$VPM_BASE_URL/api/v1/proxies?per_page=1"
# Expected: 200
```

Coverage:
- `TestClient_CreateProxy_Success` — auth header + response mapping
- `TestClient_CreateProxy_APIError_422` — error handling
- `TestClient_CreateProxy_Retries_On_5xx` — 3 retries exponential backoff
- `TestClient_DeleteProxy_Success` — DELETE path
- `TestClient_GetProxy_Success` / `NotFound`
- `TestAdapter_Purchase_ReturnsSyncCredentials` — credentials không nil
- `TestAdapter_Purchase_DefaultsToSocks5` — default protocol
- `TestAdapter_Purchase_MetadataBandwidthAndSpeed` — metadata parsing
- `TestAdapter_Cancel_CallsDelete` — DELETE được gọi đúng
- `TestAdapter_CheckStatus_Mapping` — 5 status mapping cases
- `TestAdapter_Purchase_ProviderError_MapsToInvalidConfig` — error wrapping
