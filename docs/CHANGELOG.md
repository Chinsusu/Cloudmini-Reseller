# Changelog — ProxyVPS Platform

All notable changes to this project are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versioning follows [Semantic Versioning](https://semver.org/).

---

## [Unreleased]

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
