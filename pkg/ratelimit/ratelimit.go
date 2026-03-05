// Package ratelimit provides a Redis-backed sliding-window rate limiter
// implemented as an HTTP middleware for all PVP services.
//
// Algorithm: Lua script using Redis ZADD + ZREMRANGEBYSCORE for a sliding
// window — O(log N) per request, no thundering herd on window boundary.
package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config defines rate limit parameters.
type Config struct {
	// Requests is the maximum number of requests allowed per Window.
	Requests int
	// Window is the sliding window duration (e.g. 1*time.Minute).
	Window time.Duration
	// KeyFunc extracts the rate-limit key from a request (e.g. IP or user ID).
	KeyFunc func(r *http.Request) string
}

// DefaultIPKeyFunc uses the remote IP as the rate limit key.
func DefaultIPKeyFunc(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}

// UserIDKeyFunc uses the authenticated user ID (from X-User-ID header set by middleware).
func UserIDKeyFunc(r *http.Request) string {
	uid := r.Header.Get("X-User-ID")
	if uid == "" {
		return DefaultIPKeyFunc(r)
	}
	return "user:" + uid
}

// lua script: sliding-window counter using a sorted set.
// KEYS[1] = rate limit key
// ARGV[1] = now (unix nano string)
// ARGV[2] = window start (unix nano string)
// ARGV[3] = TTL seconds
// Returns current count after adding this request.
var slidingWindowScript = redis.NewScript(`
local key    = KEYS[1]
local now    = tonumber(ARGV[1])
local winStart = tonumber(ARGV[2])
local ttl    = tonumber(ARGV[3])

-- Remove expired entries
redis.call('ZREMRANGEBYSCORE', key, '-inf', winStart)
-- Add current request (score=timestamp, member=timestamp+random suffix via counter)
redis.call('ZADD', key, now, tostring(now))
-- Set TTL to avoid orphan keys
redis.call('EXPIRE', key, ttl)
-- Return count in window
return redis.call('ZCARD', key)
`)

// Middleware returns an HTTP middleware that enforces rate limiting using Redis.
func Middleware(rdb *redis.Client, cfg Config, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := "pvp:rl:" + cfg.KeyFunc(r)
			now := time.Now().UnixNano()
			winStart := time.Now().Add(-cfg.Window).UnixNano()
			ttlSec := int(cfg.Window.Seconds()) + 1

			count, err := slidingWindowScript.Run(r.Context(), rdb,
				[]string{key},
				strconv.FormatInt(now, 10),
				strconv.FormatInt(winStart, 10),
				ttlSec,
			).Int()

			if err != nil {
				// Redis error: allow request (fail open)
				logger.Warn("rate limiter redis error", slog.String("error", err.Error()))
				next.ServeHTTP(w, r)
				return
			}

			// Set standard rate-limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.Requests))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(max(0, cfg.Requests-count)))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(cfg.Window).Unix(), 10))

			if count > cfg.Requests {
				w.Header().Set("Retry-After", strconv.Itoa(int(cfg.Window.Seconds())))
				http.Error(w, `{"error":"too_many_requests","message":"Rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Standard rate limit tiers.
var (
	// AuthTier: 20 requests / minute (login, refresh)
	AuthTier = Config{Requests: 20, Window: time.Minute, KeyFunc: DefaultIPKeyFunc}
	// APITier: 300 requests / minute per user
	APITier = Config{Requests: 300, Window: time.Minute, KeyFunc: UserIDKeyFunc}
	// AdminTier: 1000 requests / minute
	AdminTier = Config{Requests: 1000, Window: time.Minute, KeyFunc: UserIDKeyFunc}
	// StrictTier: 5 requests / minute (e.g. payment creation)
	StrictTier = Config{Requests: 5, Window: time.Minute, KeyFunc: UserIDKeyFunc}
)

// MiddlewareTier returns a rate-limit middleware for a named tier.
func MiddlewareTier(rdb *redis.Client, tier Config, logger *slog.Logger) func(http.Handler) http.Handler {
	return Middleware(rdb, tier, logger)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// go.mod helper — returns module path.
func ModulePath() string { return "github.com/pvp/ratelimit" }

// NewClient creates a Redis client for the rate limiter.
func NewClient(redisURL string) (*redis.Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("ratelimit.NewClient: %w", err)
	}
	rdb := redis.NewClient(opt)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ratelimit.NewClient: ping: %w", err)
	}
	return rdb, nil
}
