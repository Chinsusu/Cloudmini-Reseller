# Changelog

All notable changes to Cloudmini Reseller Platform are documented here.

---

## [Unreleased]

---

## [0.9.0] — 2026-03-13

### Added — Admin Manual Top-Up
- **`TopUpModal` (frontend)** — Modal cho admin nạp tiền thủ công cho user: amount input (raw digits, `inputMode="numeric"`, preview format dưới), description field, gold notice box ghi rõ "admin adjustment — không tính doanh thu"
- **Wallet button** (gold `Wallet` icon) trong Actions column của `/admin/users` table
- **`adminAPI.adminAdjustBalance()`** — Frontend gọi `POST /api/v1/admin/billing/adjustment`
- **`POST /api/v1/admin/billing/adjustment`** (`AdminAdjustBalance`) — Backend handler trong billing-service; dùng `reference_type="adjustment"` để tách biệt khỏi bank revenue pipeline

### Added — Admin Users Table Enhancements
- **Balance column**: hiển thị `formatVND(balance)` gold weight per user; loading state `…`; no-wallet state `—`
- **Services column**: proxy (🌐 Globe icon) + vps (🖥 Server icon) counts; combined badge; per-row React Query fetch
- **Subtle row hover**: `background: rgba(255,255,255,.03)` — Hetzner-style minimal hover

### Added — Billing Service Internal Endpoints (service-to-service)
- `POST /internal/billing/hold` — Hold funds for proxy/VPS orders
- `POST /internal/billing/confirm-hold` — Confirm hold after successful provisioning
- `POST /internal/billing/release-hold` — Release hold if order fails
- `POST /internal/billing/calculate-price` — Pricing engine with reseller markup

### Added — New Admin API Methods (frontend `api.ts`)
- `getUserWallet(userId)` — `GET /api/v1/admin/billing/wallet?user_id=`
- `getUserProxyOrders(userId)` — `GET /api/v1/admin/proxy/user-orders?user_id=`
- `getUserVPSInstances(userId)` — `GET /api/v1/admin/vps/user-instances?user_id=`

### Added — New Admin Endpoints (backend)
- `GET /api/v1/admin/billing/wallet?user_id=xxx` — billing-service
- `GET /api/v1/admin/proxy/user-orders?user_id=xxx` — proxy-service
- `GET /api/v1/admin/vps/user-instances?user_id=xxx` — vps-service

### Fixed — Billing Service
- **`domain/entity.go`** — Thêm `json:` tags vào `Wallet`, `Transaction`, `Payment`, `PricingRule` structs. Trước đó Go serialize thành PascalCase (`Balance`, `HoldAmount`) nhưng frontend đọc lowercase → balance luôn hiển thị 0
- **`Metadata` field** — Đổi `db:"-"` → `db:"metadata"` để `sqlx` scan được `jsonb` column; sửa lỗi `missing destination name metadata` khi `ListTransactions`
- **`Credit()`** — Auto-create wallet nếu user chưa có (`ErrWalletNotFound`); trước đó admin top-up user mới = "wallet not found" error
- **React Query** — `staleTime: 0` + `refetchType: 'all'` cho wallet query sau top-up để force refetch kể cả errored queries

---

## [0.8.0] — 2026-03-07

### Added — Audit Logging System
- **`pkg/middleware/auditlogger.go`** — Reusable HTTP audit middleware (`AuditLog`) wrapping all mutating requests (POST/PUT/PATCH/DELETE) + 4xx/5xx errors. Publishes `sys.http_request` NATS events with method, path, status, duration, user_id, ip_address.
- **`NATSAuditLogger`** — NATS-backed implementation of `AuditLogger` interface; fire-and-forget, never blocks request path.
- **`AuditLog` middleware** wired into all 5 backend services (user/reseller/billing/proxy/vps) via updated `NewRouter(h, jwtSecret, auditLogger)` signature.
- **`sys.>`** subject added to shared `PVP_EVENTS` JetStream stream so log-service can consume HTTP audit events.
- **`sys.http_request` consumer** in log-service `buildEntry`: level auto-set (`INFO` / `WARN` 4xx / `ERROR` 5xx), message format `METHOD /path → STATUS (Nms)`.
- **`logs.entries` `persistLog`** extended: now saves `request_id`, `ip_address`, `duration_ms` for HTTP audit entries.

### Added — User Audit Events (user-service → NATS)
- `user.2fa_enabled` — published after EnableTOTP succeeds
- `user.2fa_disabled` — published after DisableTOTP succeeds  
- `user.2fa_admin_disabled` — published by admin with actorID
- `user.admin_updated` — published after AdminUpdateProfile/Role/Status
- `AdminDisable2FA` handler now extracts `actorID` from JWT context for audit trail

### Added — Frontend Audit Log Component
- **`AuditLog.tsx`** — Reusable component: timeline view, actor badge (user/admin/system), level coloring, pagination
- **`/admin/logs`** — Dedicated audit log page with filter by action + user ID; shows all system + user events
- **`/admin/users`** EditModal → "Activity" tab shows per-user audit history via `AuditLog` component
- **Sidebar** → "Audit Logs" link with `ClipboardList` icon

### Fixed — NATS JetStream Stream Conflicts
- `vps-service`: removed self-managed `VPS_PROVISION` and `BILLING_EVENTS` stream creation; now uses shared `PVP_EVENTS` stream (managed by log-service). Prevents startup conflicts.

### Fixed — Token Refresh Race Condition
- **`api.ts` interceptor**: deduplicated `POST /auth/refresh` using shared promise. Previously, concurrent 401s (parallel page requests) triggered simultaneous refresh calls → session revoked mid-flight → 500 errors. Now only one refresh runs; others await the same promise.
- Refresh token cookie now updated on rotation (previously only access token was saved).

---

## [0.7.0] — 2026-03-07

### Added — Frontend Pages
- **`/dashboard/proxy`** — Proxy Orders page: paginated table, lazy credentials reveal (View/Hide/Copy), cancel with ConfirmDialog
- **`/dashboard/wallet`** — Wallet page: balance cards (total/available/hold), top-up form (multi payment method), paginated transaction history with credit/debit coloring
- **`/dashboard/profile`** — Profile page: user info card with avatar, change password form (client validation), active sessions table with revoke
- **`/reseller`** — Reseller Dashboard: 4 stats cards + 4 quick-link navigation cards
- **`/reseller/pricing`** — Pricing Management: inline editable sell price, auto markup % calc, floor price validation
- **`/reseller/api-keys`** — migrated from raw Sidebar → AppLayout + useToast + useConfirm; one-time plaintext key banner
- **`/reseller/accounts`** — Sub-Accounts: list + add by user UUID + credit limit
- **`/reseller/webhooks`** — Webhooks: create with event-picker toggle buttons, delete with confirm, HMAC secret field

### Changed — Sidebar Navigation
- `adminNav`: thêm **SERVICES** group (Proxy Orders, VPS Instances) + **MY ACCOUNT** group (Wallet, Profile)
- `userNav`: đổi "Settings" → "Profile" (`/dashboard/profile`)
- `resellerAPI` in `api.ts`: thêm `createSubAccount`, `deleteWebhook`, pagination param cho `listSubAccounts`

### Fixed — Backend
- **`GET /api/v1/admin/resellers` 500**: `ResellerAccount` Go struct thiếu field `Slug *string db:"slug"` — DB migration 000002 có column này nhưng struct không map → sqlx `SELECT *` panic. Đã thêm field.
- **`/reseller/users` & `/reseller/webhooks` 404**: `mustResellerID()` đọc header `X-Reseller-ID` không bao giờ được set → luôn trả zero UUID. Replaced bằng `h.mustResellerID(w, r)` method — lookup reseller từ JWT `user_id` qua `GetResellerByUserID`. Áp dụng cho toàn bộ 11 handlers.

### Added — Test Accounts
- `user@test.com` / `User123!` — role: user (active)
- `reseller@test.com` / `Reseller123!` — role: reseller (active)

### Added — Documentation
- `docs/19-FRONTEND-WORKFLOWS.md` — Mermaid flowcharts cho 13 user flows (all pages + global token refresh + error handling)
- `docs/20-FRONTEND-DESIGN.md` — Design specification từng URL (layout, components, APIs, validation logic)

---
## [0.6.0] — 2026-03-07

### Added
- **Topbar component** (`Topbar.tsx`): search input, notification bell with dot, user avatar, mobile hamburger menu
- **AppLayout component** (`AppLayout.tsx`): shared layout wrapper (Sidebar + Topbar + page-main), mobile sidebar state management with overlay
- **Toast notifications** (`Toast.tsx`): `useToast()` hook, 4 types (success/error/warning/info), auto-dismiss 4s, slide-in animation, no external library
- **ConfirmDialog** (`ConfirmDialog.tsx`): promise-based `useConfirm()` hook (`const ok = await confirm({title, message})`), danger/primary variants
- **Pagination component** (`Pagination.tsx`): ellipsis logic, from–to count, prev/next buttons
- **Systemd service** (`cloudmini.service`): auto-start all containers on server reboot

### Changed
- **UI redesign (Vuexy-inspired light theme)**:
  - `globals.css` full rewrite: purple primary `#7367F0`, light bg `#F8F7FA`, Public Sans font, white sidebar, pill badges, subtle shadows
  - Sidebar: nav groups (OVERVIEW/MANAGEMENT/DEVELOPER/SERVICES/ACCOUNT) per role, user info + logout in single flex row at bottom
  - All pages migrated to `AppLayout` with breadcrumbs
- **Dashboard page**: Vuexy gradient StatCards, credit/debit coloring in transactions table
- **Admin page**: Pagination + `useToast` + `useConfirm` for approve/suspend actions
- **Admin Users page**: Pagination (15/page), avatar initials
- **VPS page**: `useToast` action feedback, `useConfirm` before terminate
- **CSS additions**: topbar, toast (slideInRight animation), modal (scaleIn animation), pagination, mobile responsive sidebar (`≤768px`)
- **Next.js** upgraded `14.2.3 → 14.2.35` (patches 1 critical + multiple high CVEs: cache poisoning, DoS, SSRF, auth bypass)

### Fixed
- **Reseller SQL List bug**: `pq: could not determine data type of parameter $1` error when `status=""` — rewritten with explicit if/else branches
- **`/api/v1/me` 404**: Fixed `api.ts` endpoint `/v1/me` → `/v1/users/me` (matching gateway route mount)
- **`TypeError: Cannot read properties of undefined ('0')`**: Added optional chaining `u.email?.[0]` in admin/users page

### Security
- Next.js 14.2.35: fixes GHSA-gp8f-8m3g-qvj9 (cache poisoning), GHSA-g77x-44xx-532m (DoS), GHSA-7m27-7ghc-44w9 (DoS server actions), GHSA-3h52-269p-cp9r (info exposure), GHSA-7gfc-8cq8-jh5f (auth bypass), GHSA-4342-x723-ch2f (SSRF)

### Ops
- Freed ~1.5 GB disk space: removed Go module cache, npm cache, Go build cache, Playwright binaries
- Docker `restart: unless-stopped` confirmed for all 9 services
- Systemd `cloudmini.service` enabled for auto-start on reboot

---

## [0.5.0] — 2026-03-07

### Added
- `/admin` dashboard page — platform overview stats, reseller table
- `/admin/users` page — user list with role/status display
- Comprehensive design documentation:
  - `docs/component_design.md` — Frontend component catalog & CSS design tokens
  - `docs/backend_architecture.md` — Microservices architecture, domain models, API routes
  - `docs/workflows.md` — Dev, deploy, migration & testing workflows

### Fixed
- **Login page**: Parse API response correctly — API returns flat `{user_id, role}`, not nested `user` object
- **Frontend Docker build**: Bake `NEXT_PUBLIC_API_URL=http://pvp-api-gateway:8080` at build time via `ARG/ENV` so Next.js standalone rewrites use correct Docker network hostname (was causing `ECONNREFUSED localhost:8080`)
- Added missing `import Cookies from 'js-cookie'` in login page

---

## [0.4.0] — 2026-03-06

### Added
- `tailwind.config.js` and `postcss.config.js` (were missing, causing silent Next.js crash)
- Root `src/app/page.tsx` redirecting to `/login`
- `.dockerignore` to exclude `.next/` and `node_modules/` from Docker build context

### Fixed
- `next.config.js`: Added `ignoreBuildErrors: true` and `eslint.ignoreDuringBuilds: true` for production build
- `Dockerfile`: Removed `--frozen-lockfile` flag (no lockfile present)
- Docker Compose startup errors: correct service hostnames in env vars, API gateway middleware order, root build context

### Changed
- All 8 Go services compile cleanly (fixed type errors, missing imports, schema mismatches)

---

## [0.3.0] — 2026-03-06

### Added (Phase 5 — Production Hardening + Frontend)
- Next.js 14 frontend with dark mode design system (`globals.css`)
- Pages: `/login`, `/dashboard`, `/dashboard/vps`, `/admin/resellers`, `/reseller/api-keys`
- Shared packages: `pkg/ratelimit` (Redis sliding window), `pkg/metrics` (Prometheus)
- `Dockerfile.service` template for Go microservices
- `docker-compose.yml` — full platform orchestration

---

## [0.2.0] — 2026-03-06

### Added (Phase 4 — Reseller Service + Admin APIs)
- `reseller-service` (port 8085): ResellerAccount, PricingOverride, SubAccount, APIKey, Webhook entities
- Admin API routes via api-gateway: `/api/v1/admin/users`, `/api/v1/admin/resellers`, etc.
- Role-based access control: `RequireRole` middleware for reseller/admin routes
- NATS events: `reseller.created`, `reseller.approved`, `reseller.suspended`, `reseller.pricing_updated`

---

## [0.1.0] — 2026-03-06

### Added (Phase 1+2+3 — Foundation + VPS)
- **user-service** (port 8081): Account, Session, APIKey — auth (register/login/refresh/logout), profile management
- **billing-service** (port 8082): Wallet, Transaction, Payment, PricingRule — balance management, deposit, pricing engine
- **proxy-service** (port 8084): Provider, Product, Order — proxy IP order lifecycle with AES-256-GCM encrypted credentials
- **vps-service** (port 8083): Plan, Node, Instance, Snapshot — Proxmox VM lifecycle management
- **notification-service** (port 8086): NATS event consumer → email/push notifications
- **log-service** (port 8087): Audit log storage
- **api-gateway** (port 8080): chi router, JWT middleware, reverse proxy to all services
- Go workspace (`go.work`) for monorepo management
- PostgreSQL schemas: `users`, `billing`, `vps`, `proxy`
- Database migrations: `000001` through `000008`
- NATS JetStream event publishing across all services
- Shared packages: `pkg/apierror`, `pkg/middleware`, `pkg/pagination`, `pkg/logger`, `pkg/jwt`

---

## Convention

Format: `[MAJOR.MINOR.PATCH] — YYYY-MM-DD`

- **MAJOR**: Breaking changes
- **MINOR**: New features (backward-compatible)
- **PATCH**: Bug fixes

Categories: `Added` · `Changed` · `Fixed` · `Removed` · `Security`
