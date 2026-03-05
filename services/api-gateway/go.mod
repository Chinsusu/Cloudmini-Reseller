module github.com/pvp/api-gateway

go 1.22

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/redis/go-redis/v9 v9.6.1
	github.com/pvp/pkg/logger v0.0.0
	github.com/pvp/pkg/middleware v0.0.0
	github.com/pvp/pkg/apierror v0.0.0
)

replace (
	github.com/pvp/pkg/logger => ../../pkg/logger
	github.com/pvp/pkg/middleware => ../../pkg/middleware
	github.com/pvp/pkg/apierror => ../../pkg/apierror
)
