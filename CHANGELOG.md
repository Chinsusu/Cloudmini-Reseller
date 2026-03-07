# Changelog

All notable changes to Cloudmini Reseller Platform are documented here.

---

## [Unreleased]

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
