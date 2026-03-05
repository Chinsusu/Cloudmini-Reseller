# 🚀 ProxyVPS Platform

> Enterprise-grade Proxy & VPS reselling platform with multi-tier reseller support, real-time logging, and Proxmox integration.

---

## 📋 Project Info

| Field | Value |
|---|---|
| **Project Code** | PVP-2025 |
| **Version** | 1.0.0 |
| **Status** | In Development |
| **Owner** | Platform Team |
| **Language** | Go 1.22+ |
| **License** | Proprietary |

---

## 🗂️ Documentation Index

### Core
| Doc | Description |
|---|---|
| [Project Overview](docs/00-PROJECT-OVERVIEW.md) | Goals, scope, stakeholders, constraints |
| [System Architecture](docs/01-ARCHITECTURE.md) | High-level design, service map, data flow |
| [Database Design](docs/02-DATABASE-DESIGN.md) | Full schema, ERD, indexing strategy |

### Services
| Doc | Description |
|---|---|
| [API Gateway](docs/services/03-API-GATEWAY.md) | Routing, auth middleware, rate limiting |
| [User & Auth Service](docs/services/04-USER-AUTH-SERVICE.md) | Registration, JWT, session management |
| [Proxy Service](docs/services/05-PROXY-SERVICE.md) | Provider abstraction, order lifecycle |
| [VPS Service](docs/services/06-VPS-SERVICE.md) | VM provisioning, lifecycle management |
| [Billing Service](docs/services/07-BILLING-SERVICE.md) | Wallet, pricing, metering, invoices |
| [Log Service](docs/services/08-LOG-SERVICE.md) | Event tracking, real-time WebSocket |
| [Notification Service](docs/services/09-NOTIFICATION-SERVICE.md) | Email, webhook, alert triggers |
| [Reseller Service](docs/services/10-RESELLER-SERVICE.md) | Tier management, custom pricing, sub-accounts |

### Infrastructure
| Doc | Description |
|---|---|
| [Proxmox Adapter](docs/infrastructure/11-PROXMOX-ADAPTER.md) | Node management, VM operations, failover |
| [Provider Adapters](docs/infrastructure/12-PROVIDER-ADAPTERS.md) | Proxy provider abstraction layer |

### Standards
| Doc | Description |
|---|---|
| [Go Coding Standard](docs/standards/13-CODING-STANDARD-GO.md) | Code style, patterns, review checklist |
| [API Design Standard](docs/standards/14-API-DESIGN-STANDARD.md) | REST conventions, versioning, error format |
| [Commit Convention](docs/standards/15-COMMIT-CONVENTION.md) | Conventional commits, scope rules |
| [Git Workflow](docs/standards/16-GIT-WORKFLOW.md) | Branch strategy, PR process, release |

### AI Development
| Doc | Description |
|---|---|
| [AI Prompts — Gemini](docs/ai/17-AI-PROMPT-GEMINI.md) | Prompt templates for Google Gemini / AI Studio |
| [Project Context Workflow](docs/ai/18-PROJECT-CONTEXT-WORKFLOW.md) | How to generate & maintain AI context |

---

## ⚡ Quick Start

```bash
# Clone repo
git clone git@github.com:org/proxy-vps-platform.git
cd proxy-vps-platform

# Setup environment
cp .env.example .env

# Start infrastructure
docker compose up -d postgres redis nats

# Run all services (dev mode)
make dev

# Run tests
make test
```

---

## 🏗️ Tech Stack

- **Backend**: Go 1.22 (microservices)
- **Message Bus**: NATS JetStream
- **Database**: PostgreSQL 16 + Redis 7
- **Infrastructure**: Proxmox VE 8.x
- **Frontend**: Next.js 14
- **Deployment**: Docker + Docker Compose / Kubernetes
- **Monitoring**: Prometheus + Grafana
- **CI/CD**: GitHub Actions

---

## 📁 Repository Structure

```
proxy-vps-platform/
├── services/
│   ├── api-gateway/
│   ├── user-service/
│   ├── proxy-service/
│   ├── vps-service/
│   ├── billing-service/
│   ├── log-service/
│   ├── notification-service/
│   └── reseller-service/
├── pkg/                    # shared packages
│   ├── nats/
│   ├── postgres/
│   ├── logger/
│   └── middleware/
├── infrastructure/
│   ├── proxmox/
│   └── providers/
├── frontend/
├── deploy/
│   ├── docker/
│   └── k8s/
├── docs/
├── scripts/
├── Makefile
└── docker-compose.yml
```

---

## 📌 Changelog

See [CHANGELOG.md](CHANGELOG.md)
