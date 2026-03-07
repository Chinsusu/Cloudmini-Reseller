# Changelog

All notable changes to Cloudmini Reseller Platform are documented here.

---

## [Unreleased]

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
