.PHONY: help dev build-all test-all lint fmt migrate-up migrate-down infra-up infra-down monitoring-up frontend-dev

SERVICES := api-gateway user-service proxy-service vps-service billing-service log-service notification-service reseller-service
PKGS    := nats postgres logger middleware apierror crypto pagination ratelimit circuitbreaker retry metrics

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ─── Infrastructure ──────────────────────────────────────────────────────────
infra-up: ## Start only infrastructure (postgres, redis, nats)
	docker compose up -d postgres redis nats
	@echo "Waiting for infra to be healthy..."
	@sleep 3

infra-down: ## Stop infrastructure
	docker compose down

# ─── Dev ─────────────────────────────────────────────────────────────────────
dev: infra-up migrate-up ## Start all services in dev mode (air hot reload)
	@for svc in $(SERVICES); do \
		echo "Starting $$svc..."; \
		(cd services/$$svc && air &); \
	done; \
	wait

# ─── Build ───────────────────────────────────────────────────────────────────
build-all: ## Build all services
	@for svc in $(SERVICES); do \
		echo "Building $$svc..."; \
		(cd services/$$svc && go build ./...) || exit 1; \
	done

build-%: ## Build a specific service: make build-user-service
	cd services/$* && go build ./...

# ─── Test ────────────────────────────────────────────────────────────────────
test-all: ## Run all unit tests
	@for svc in $(SERVICES); do \
		echo "Testing $$svc..."; \
		(cd services/$$svc && go test ./... -race -count=1) || exit 1; \
	done
	@for pkg in nats postgres logger middleware apierror crypto pagination; do \
		(cd pkg/$$pkg && go test ./... -race -count=1) || exit 1; \
	done

test-%: ## Test a specific service: make test-user-service
	cd services/$* && go test ./... -race -count=1 -v

test-integration: infra-up migrate-up ## Run integration tests
	go test ./... -tags=integration -race -v

# ─── Lint ────────────────────────────────────────────────────────────────────
lint: ## Run golangci-lint on all services
	@for svc in $(SERVICES); do \
		echo "Linting $$svc..."; \
		(cd services/$$svc && golangci-lint run ./...) || exit 1; \
	done

fmt: ## Format all Go code
	go fmt ./...
	goimports -w .

# ─── Migration ───────────────────────────────────────────────────────────────
migrate-up: ## Apply all migrations
	migrate -path migrations -database "$$DATABASE_URL" up

migrate-down: ## Rollback last migration
	migrate -path migrations -database "$$DATABASE_URL" down 1

migrate-status: ## Show migration status
	migrate -path migrations -database "$$DATABASE_URL" version

# ─── Docker ──────────────────────────────────────────────────────────────────
docker-up: ## Start all services with Docker Compose
	docker compose up --build -d

docker-down: ## Stop all Docker services
	docker compose down -v

docker-logs: ## Follow logs for all services
	docker compose logs -f

# ─── Utility ─────────────────────────────────────────────────────────────────
tidy-all: ## go mod tidy for all modules
	@for svc in $(SERVICES); do \
		(cd services/$$svc && go mod tidy); \
	done
	@for pkg in $(PKGS); do \
		(cd pkg/$$pkg && go mod tidy); \
	done
	(cd infrastructure/proxmox && go mod tidy)
	(cd infrastructure/providers && go mod tidy)

# ─── Monitoring ───────────────────────────────────────────────────────────────
monitoring-up: ## Start monitoring stack (Prometheus + Grafana)
	docker compose -f docker-compose.yml -f docker-compose.monitoring.yml up -d prometheus grafana alertmanager postgres-exporter redis-exporter
	@echo "Grafana:    http://localhost:3001  (admin / pvpadmin)"
	@echo "Prometheus: http://localhost:9090"

monitoring-down: ## Stop monitoring stack
	docker compose -f docker-compose.yml -f docker-compose.monitoring.yml down prometheus grafana alertmanager postgres-exporter redis-exporter

# ─── Frontend ─────────────────────────────────────────────────────────────────
frontend-dev: ## Run Next.js frontend in dev mode
	cd frontend && npm install && npm run dev

frontend-build: ## Build Next.js frontend
	cd frontend && npm install && npm run build
