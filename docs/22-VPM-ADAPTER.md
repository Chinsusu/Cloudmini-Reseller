# VPM Provider Adapter — Design Document

**Document ID**: PVP-DOC-022  
**Version**: 3.0.0  
**Component**: `proxy-service / providers / vpm`  
**Last Updated**: 2026-03-26

---

## 1. Overview

**VPS Proxy Manager (VPM)** là internal service quản lý proxy trực tiếp trên các node vật lý. Cloudmini tích hợp VPM dưới dạng một provider adapter (`vpm`), implement `IProxyProvider` interface.

- **Base URL**: `https://cz.resvn.net` (production)
- **Authentication**: `X-API-Key` header (primary) + `?access_code=` query param (compat)
- **API Docs**: `https://cz.resvn.net/billing-docs-v1`
- **Provider type**: **Async** — `POST /api/v1/proxies` returns `status: "creating"`, adapter polls until `"running"`

---

## 2. Adapter Structure

```
services/proxy-service/internal/providers/vpm/
├── types.go        # DTOs: CreateProxyRequest, ProxySummary, CheckResult, APIError
├── client.go       # HTTP client: X-API-Key auth, retry 3x, async polling, envelope unwrap
├── adapter.go      # IProxyProvider impl: Purchase/Cancel/Suspend/Resume/CheckStatus
└── adapter_test.go # Unit tests
```

---

## 3. Interface Implementation

| Method | VPM API V1 Endpoint | Behavior |
|---|---|---|
| `Purchase` | `POST /api/v1/proxies` | Async — polls until `running` (max 60s) |
| `Cancel` | `DELETE /api/v1/proxies/{id}` | Permanent — 204 No Content |
| `Suspend` | `POST /api/v1/proxies/{id}/stop` | Temporary — traffic disabled |
| `Resume` | `POST /api/v1/proxies/{id}/start` | Restore traffic |
| `CheckStatus` | `GET /api/v1/proxies/{id}` | Map status → Cloudmini constants |

---

## 4. VPM API V1 Endpoints

### 4.1 Create Proxy

```http
POST /api/v1/proxies
X-API-Key: <key>
Content-Type: application/json

{
  "protocol": "default",
  "group_id": "8fa2b3c4-...",     // optional: region UUID
  "listen_ip": "103.151.53.41",   // optional: bind to specific IP
  "ip_address_id": "...",         // optional: specific IP UUID
  "ip_range_id": "...",           // optional: specific IP range
  "bandwidth_limit_mb": 0,       // 0 = unlimited
  "speed_limit_mbps": 0
}
```

**Response** (201 Created):
```json
{
  "success": true,
  "data": {
    "id": "550e8400-...",
    "host": "103.151.53.41",
    "port": 27642,
    "port_http": 58668,
    "port_socks": 27642,
    "protocol": "default",
    "outbound_ip": "103.151.53.41",
    "auth_user": "u_abc12345",
    "auth_pass": "SomePass16chars",
    "status": "creating",
    "connect_url": "socks5://u_abc12345:pass@103.151.53.41:27642"
  }
}
```

> **Note**: Status starts as `"creating"`. Poll `GET /api/v1/proxies/:id` until `"running"`.

### 4.2 Supported Protocols

| Protocol | port_http | port_socks | Credentials |
|---|---|---|---|
| `default` | ✅ | ✅ | 2 credentials (HTTP + SOCKS5) |
| `http` | ✅ | ❌ | 1 credential |
| `socks5` | ❌ | ✅ | 1 credential |
| `vmess` | — | — | `connect_url` |
| `vless` | — | — | `connect_url` |
| `shadowsocks` | — | — | `connect_url`, `ss_method` |
| `trojan` | — | — | `connect_url` |

### 4.3 Get Proxy Detail

```http
GET /api/v1/proxies/{id}
X-API-Key: <key>
```

### 4.4 Delete Proxy

```http
DELETE /api/v1/proxies/{id}
X-API-Key: <key>
```
Response: `204 No Content`

### 4.5 Stop Proxy

```http
POST /api/v1/proxies/{id}/stop
X-API-Key: <key>
```
Response: `{"success": true, "data": "proxy stopped"}`

### 4.6 Start Proxy

```http
POST /api/v1/proxies/{id}/start
X-API-Key: <key>
```
Response: `{"success": true, "data": "proxy started"}`

### 4.7 Check Proxy IP

```http
GET /api/v1/proxies/{id}/check
X-API-Key: <key>
```
Response:
```json
{
  "success": true,
  "data": {
    "ip": "103.151.53.41",
    "country": "VN",
    "city": "Ho Chi Minh City",
    "org": "AS135918 VIET DIGITAL...",
    "latency_ms": 361
  }
}
```

### 4.8 List Groups

```http
GET /api/v1/groups
X-API-Key: <key>
```

---

## 5. Status Mapping

| VPM Status | Cloudmini Status | Notes |
|---|---|---|
| `running` | `active` | Normal operation |
| `stopped` | `active` | Proxy reserved, temporarily paused |
| `creating` | `processing` | Async creation in progress |
| `error` | `failed` | Provisioning failed |

---

## 6. Async Purchase Flow

```
Frontend → POST /proxy/orders → OrderUsecase.CreateOrder()
  → adapter.Purchase()
    → client.CreateProxyAndWait()
      → POST /api/v1/proxies → status: "creating"
      → poll GET /api/v1/proxies/:id every 2s (max 60s)
      → status: "running" → return ProxySummary
    → buildCredentials() → return PurchaseResult
  → encrypt credentials → update order → active
```

---

## 7. Error Codes

| VPM Error Code | Meaning |
|---|---|
| `INVALID_INPUT` | Bad request parameters |
| `NO_AVAILABLE_IP` | No free IPs in group |
| `NODE_AT_CAPACITY` | Node full |
| `PROXY_NOT_FOUND` | ID doesn't exist |
| `INTERNAL_ERROR` | VPM internal failure |

---

## 8. Changelog

| Date | Version | Changes |
|---|---|---|
| 2026-03-26 | 3.0.0 | **Migrate to Billing API V1**: async creation flow, X-API-Key auth, field renames (auth_user/auth_pass/host), stop/start replaces lock/unlock, added CheckProxy, removed pipe-separated IDs |
| 2026-03-24 | 2.0.0 | Migrate to API v2, dual-port default protocol, lock/unlock by IP |
| 2026-03-22 | 1.0.0 | Initial VPM adapter (API v1 legacy) |
