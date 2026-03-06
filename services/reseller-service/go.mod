module github.com/pvp/reseller-service

go 1.22

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/google/uuid v1.6.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/pvp/pkg/apierror v0.0.0
	github.com/pvp/pkg/logger v0.0.0
	github.com/pvp/pkg/middleware v0.0.0
	github.com/pvp/pkg/nats v0.0.0
	github.com/pvp/pkg/pagination v0.0.0
	github.com/pvp/pkg/postgres v0.0.0
	github.com/shopspring/decimal v1.4.0
)

require (
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/golang-migrate/migrate/v4 v4.17.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/klauspost/compress v1.17.2 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/nats-io/nats.go v1.36.0 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	golang.org/x/crypto v0.20.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace (
	github.com/pvp/pkg/apierror => ../../pkg/apierror
	github.com/pvp/pkg/logger => ../../pkg/logger
	github.com/pvp/pkg/middleware => ../../pkg/middleware
	github.com/pvp/pkg/nats => ../../pkg/nats
	github.com/pvp/pkg/pagination => ../../pkg/pagination
	github.com/pvp/pkg/postgres => ../../pkg/postgres
)
