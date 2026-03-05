module github.com/pvp/providers

go 1.22

require (
	github.com/google/uuid v1.6.0
	github.com/pvp/pkg/logger v0.0.0
)

replace (
	github.com/pvp/pkg/logger => ../../pkg/logger
)
