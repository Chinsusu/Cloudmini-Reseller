# ProxyVPS Platform

B2B/B2C platform for proxy and VPS reselling, supporting multi-tier resellers.

## Tech Stack
- **Language**: Go 1.22+
- **Database**: PostgreSQL 16
- **Messaging**: NATS JetStream
- **Cache**: Redis 7
- **VPS Backend**: Proxmox VE 8.x

## Project Structure
```
├── services/          # Microservices
│   ├── api-gateway/
│   ├── user-service/
│   ├── billing-service/
│   ├── proxy-service/
│   ├── vps-service/
│   ├── log-service/
│   ├── notification-service/
│   └── reseller-service/
├── pkg/               # Shared packages
├── migrations/        # Database migrations
├── infrastructure/    # External adapters
└── docs/              # Documentation
```

## Quick Start
```bash
cp .env.example .env
# Edit .env (set JWT_SECRET, etc.)
make infra-up
make migrate-up
cd services/user-service && go run ./cmd/server/
```

## Development
```bash
make dev        # Start all services
make test-all   # Run all tests
make lint       # Lint all services
```
