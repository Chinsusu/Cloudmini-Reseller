# Changelog — ProxyVPS Platform

All notable changes to this project are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versioning follows [Semantic Versioning](https://semver.org/).

---

## [Unreleased]

---

## [0.12.0] — 2026-03-23

### Added
- **proxy-service** Expiry lifecycle scheduler (runs every 15 minutes):
  - `ExpiryUsecase.ProcessExpired`: active orders past `COALESCE(custom_expires_at, expires_at)` →
    status `expired` + VPM `POST /proxies/{id}/stop` (suspends port; no data lost)
  - `ExpiryUsecase.ProcessGraceExpired`: expired orders past 72h grace →
    status `cancelled` + VPM `DELETE /proxies/{id}` (permanent delete, no refund)
  - Log events: `order.expired`, `order.deleted`
- **proxy-service** `IProxyProvider.Suspend(ctx, providerOrderID)` method:
  - VPM: calls `POST /proxies/{id}/stop`
  - Proxy-Cheap, Sandbox: no-op (providers có không có suspend API)
- **proxy-service** `OrderRepository.ListExpiredActive` + `ListExpiredGrace` queries —
  dùng `COALESCE(custom_expires_at, expires_at)` để tôn trọng admin override

### Fixed
- **frontend** `EditOrderModal`: giá không còn bị hiện strikethrough sai khi admin chỉ sửa expiry
  - Before: `price` state init từ `unit_price` → `custom_price` được gửi lên = unit_price → strikethrough
  - After: `price` state init từ `custom_price ?? ''` → chỉ gửi nếu user thực sự nhập giá mới

---

## [0.11.0] — 2026-03-23

### Added
- **proxy-service** VPM provider adapter (`internal/providers/vpm/`) — tích hợp VPS Proxy Manager
  vào Cloudmini như một sync proxy provider (credentials trả về ngay, không cần webhook)
  - `types.go`: DTOs ánh xạ VPM API (`CreateProxyRequest`, `ProxySummary`, `APIError`)
  - `client.go`: HTTP client với Bearer auth, retry 3x exponential backoff, envelope unwrapping
  - `adapter.go`: Implement `IProxyProvider` (`Purchase`→`POST /proxies`, `Cancel`→`DELETE /proxies/{id}`,
    `CheckStatus`→`GET /proxies/{id}`)
  - `adapter_test.go`: 13 unit tests (httptest.NewServer, không cần server thật)
- **proxy-service** `migrations/000010_add_vpm_provider.sql` — seed `proxy.providers` row cho VPM
  với UUID `b2000000-0000-0000-0000-000000000002`
- **docs** `22-VPM-ADAPTER.md` — tài liệu thiết kế đầy đủ cho VPM adapter

### Fixed
- **proxy-service** `OrderRepository.ListByUser` exclude `status='failed'` orders khỏi danh sách
  proxy của user — failed orders chỉ lưu trong `proxy.order_events` log, không hiển thị trong UI

### Changed
- **proxy-service** VPM adapter register bằng UUID provider (`b2000000-...`) thay vì string tên,
  đảm bảo khớp với cách `order_usecase` lookup bằng `product.ProviderID.String()`

---

## [0.10.0] — 2026-03-13

### Fixed
- **vps-service** `BUG-02`: `Instance` struct missing `os_template`, `ipv6_address`, `billing_type`,
  `ip_address_str`, `request_id`, `updated_at`, `node_name`, `idempotency_key` db tags —
  caused `missing destination name os_template` 500 errors on all `/vps/instances` calls
  (`services/vps-service/internal/domain/entity.go`,
  `services/vps-service/internal/repository/postgres/vps_repo.go`,
  `services/vps-service/internal/usecase/provision_usecase.go`)
- **vps-service** `BUG-02`: Replace `SELECT *` with explicit `instanceCols` constant casting
  `ip_address::text` and `ipv6_address::text` to avoid sqlx `inet` scan errors
- **vps-service** `BUG-02`: `UpdateAfterProvisioning` now writes both `ip_address` (inet) and
  `ip_address_str` (text) to avoid driver type errors
- **vps-service** `BUG-02`: `ProvisionUsecase.CreateVPS` now populates all NOT NULL columns:
  `instance_number`, `os_template`, `billing_type`, `idempotency_key`
- **seed** `BUG-03`: Created missing `billing.wallets` rows for `user@test.com` and `reseller@test.com`
- **seed** `BUG-04`: Created missing `resellers.accounts` row for `reseller@test.com`
  (status=approved, commission=10%, slug=test-reseller)
- **frontend** `BUG-01`: Add `src/middleware.ts` Next.js route guard — reads `pvp_token` cookie,
  decodes JWT role, redirects `/admin` (admin only), `/reseller` (reseller only),
  `/dashboard` (authenticated only)
- **frontend** `BUG-05`: Fix profile page query `select` to properly unwrap backend's `{data:{...}}`
  response envelope (`(d.data as any)?.data`) so `full_name`, `email`, `role` display correctly
- **frontend** `BUG-07`: Change balance fallback from `—` to `Chưa có ví` for clearer UX
  when admin billing API returns 404 for users without a wallet row
- **reseller-service** `BUG-06`: Change `ResellerAccount` nullable fields (`email`, `phone`,
  `address`, `tax_id`, `api_key_prefix`, `notes`, `suspend_reason`) from `string` to `*string`
  to fix `converting NULL to string is unsupported` error on reseller account scan
- **reseller-service** `BUG-06`: Add `apiKeyCols` and `webhookCols` explicit column lists
  that exclude `text[]` columns (`scopes`, `events`) which sqlx cannot auto-scan with `db:"-"` tag
- **reseller-service** `BUG-06`: Add `strPtr()` helper in usecase for safe `string → *string` conversion


### Added
- Initial project documentation (full suite)
- System architecture design
- Database schema design for all services
- Service design documents for all 8 microservices
- Proxmox Adapter design
- Provider Adapters interface design
- Go Coding Standard
- API Design Standard
- Commit Convention (Conventional Commits)
- Git Workflow (GitHub Flow)
- AI Prompt Templates (Gemini)
- Project Context Workflow for AI-assisted development

---

## [1.0.0] — TBD

### Added
- User registration and email verification
- JWT authentication with refresh token rotation
- API key management
- Wallet system (prepaid model)
- Payment gateway integration (Stripe, VNPay)
- Proxy product catalog
- Proxy order creation, renewal, cancellation
- Provider adapter system with failover routing
- VPS plan catalog
- VPS instance provisioning via Proxmox API
- VM lifecycle management (start, stop, reboot, suspend, terminate)
- Snapshot management
- Hourly VPS metering and billing
- Full audit logging (all events persisted)
- Real-time WebSocket event streaming
- Email notifications (order, billing, alerts)
- Reseller account management
- Reseller custom pricing
- Reseller sub-account management
- Reseller wallet system
- Admin dashboard APIs
- Reseller webhook delivery

---

## Versioning Policy

| Change type | Version bump |
|---|---|
| Bug fixes, security patches | PATCH (x.x.1) |
| New features (backward compatible) | MINOR (x.1.0) |
| Breaking API changes | MAJOR (2.0.0) |

## Deprecation Policy

Features deprecated with notice of minimum **2 minor versions** before removal.
Deprecated endpoints return header: `Deprecated: true` and `Sunset: {date}`.
