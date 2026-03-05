// Package domain contains the core domain entities, repository interfaces,
// and error definitions for user-service.
package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ─── Entities ────────────────────────────────────────────────────────────────

// Role constants.
const (
	RoleUser      = "user"
	RoleReseller  = "reseller"
	RoleAdmin     = "admin"
	RoleSuperAdmin = "super_admin"
)

// Status constants for user accounts.
const (
	StatusActive              = "active"
	StatusSuspended           = "suspended"
	StatusBanned              = "banned"
	StatusPendingVerification = "pending_verification"
)

// Account represents a platform user account.
type Account struct {
	ID            uuid.UUID  `db:"id"`
	Email         string     `db:"email"`
	PasswordHash  string     `db:"password_hash"`
	FullName      string     `db:"full_name"`
	Phone         string     `db:"phone"`
	Role          string     `db:"role"`
	Status        string     `db:"status"`
	ResellerID    *uuid.UUID `db:"reseller_id"`
	EmailVerified bool       `db:"email_verified"`
	LastLoginAt   *time.Time `db:"last_login_at"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
	DeletedAt     *time.Time `db:"deleted_at"`
}

// Session represents a refresh token session.
type Session struct {
	ID           uuid.UUID  `db:"id"`
	UserID       uuid.UUID  `db:"user_id"`
	RefreshToken string     `db:"refresh_token"`
	IPAddress    string     `db:"ip_address"`
	UserAgent    string     `db:"user_agent"`
	ExpiresAt    time.Time  `db:"expires_at"`
	CreatedAt    time.Time  `db:"created_at"`
	RevokedAt    *time.Time `db:"revoked_at"`
}

// APIKey represents an API key belonging to a user.
type APIKey struct {
	ID         uuid.UUID  `db:"id"`
	UserID     uuid.UUID  `db:"user_id"`
	Name       string     `db:"name"`
	KeyHash    string     `db:"key_hash"`
	KeyPrefix  string     `db:"key_prefix"`
	Scopes     []string   `db:"scopes"`
	LastUsedAt *time.Time `db:"last_used_at"`
	ExpiresAt  *time.Time `db:"expires_at"`
	CreatedAt  time.Time  `db:"created_at"`
	RevokedAt  *time.Time `db:"revoked_at"`
}

// ─── Repository Interfaces ────────────────────────────────────────────────────

// IAccountRepository defines persistence operations for user accounts.
type IAccountRepository interface {
	Create(ctx context.Context, acc *Account) error
	GetByID(ctx context.Context, id uuid.UUID) (*Account, error)
	GetByEmail(ctx context.Context, email string) (*Account, error)
	Update(ctx context.Context, acc *Account) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateRole(ctx context.Context, id uuid.UUID, role string) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]*Account, int, error)
}

// ISessionRepository defines persistence operations for sessions.
type ISessionRepository interface {
	Create(ctx context.Context, s *Session) error
	GetByToken(ctx context.Context, hashedToken string) (*Session, error)
	CountActiveByUser(ctx context.Context, userID uuid.UUID) (int, error)
	RevokeByToken(ctx context.Context, hashedToken string) error
	RevokeAllByUser(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) (int64, error)
}

// IAPIKeyRepository defines persistence operations for API keys.
type IAPIKeyRepository interface {
	Create(ctx context.Context, k *APIKey) error
	GetByHash(ctx context.Context, keyHash string) (*APIKey, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*APIKey, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}

// IEventPublisher publishes NATS events.
type IEventPublisher interface {
	PublishUserRegistered(ctx context.Context, userID uuid.UUID, email string) error
	PublishUserVerified(ctx context.Context, userID uuid.UUID) error
	PublishUserLogin(ctx context.Context, userID uuid.UUID, ip string) error
	PublishPasswordChanged(ctx context.Context, userID uuid.UUID) error
	PublishUserSuspended(ctx context.Context, userID uuid.UUID, reason string) error
}
