module github.com/pvp/notification-service

go 1.22

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/google/uuid v1.6.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/lib/pq v1.10.9
	github.com/nats-io/nats.go v1.36.0
	github.com/pvp/pkg/logger v0.0.0
	github.com/pvp/pkg/postgres v0.0.0
	github.com/pvp/pkg/nats v0.0.0
	github.com/pvp/pkg/middleware v0.0.0
	github.com/pvp/pkg/apierror v0.0.0
)

replace (
	github.com/pvp/pkg/logger => ../../pkg/logger
	github.com/pvp/pkg/postgres => ../../pkg/postgres
	github.com/pvp/pkg/nats => ../../pkg/nats
	github.com/pvp/pkg/middleware => ../../pkg/middleware
	github.com/pvp/pkg/apierror => ../../pkg/apierror
)
