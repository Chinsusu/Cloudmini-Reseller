package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pvp/user-service/internal/domain"
)

// AccountRepository implements domain.IAccountRepository using PostgreSQL.
type AccountRepository struct {
	db *sqlx.DB
}

// NewAccountRepository creates an AccountRepository.
func NewAccountRepository(db *sqlx.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Create(ctx context.Context, acc *domain.Account) error {
	query := `
		INSERT INTO users.accounts
			(id, email, password_hash, full_name, phone, role, status,
			 reseller_id, email_verified, created_at, updated_at)
		VALUES
			(:id, :email, :password_hash, :full_name, :phone, :role, :status,
			 :reseller_id, :email_verified, :created_at, :updated_at)
	`
	if _, err := r.db.NamedExecContext(ctx, query, acc); err != nil {
		return fmt.Errorf("AccountRepository.Create: %w", err)
	}
	return nil
}

func (r *AccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	var acc domain.Account
	query := `SELECT * FROM users.accounts WHERE id = $1 AND deleted_at IS NULL`
	if err := r.db.GetContext(ctx, &acc, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("AccountRepository.GetByID: %w", err)
	}
	return &acc, nil
}

func (r *AccountRepository) GetByEmail(ctx context.Context, email string) (*domain.Account, error) {
	var acc domain.Account
	query := `SELECT * FROM users.accounts WHERE email = $1 AND deleted_at IS NULL`
	if err := r.db.GetContext(ctx, &acc, query, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("AccountRepository.GetByEmail: %w", err)
	}
	return &acc, nil
}

func (r *AccountRepository) Update(ctx context.Context, acc *domain.Account) error {
	acc.UpdatedAt = time.Now()
	query := `
		UPDATE users.accounts SET
			full_name = :full_name,
			phone = :phone,
			password_hash = :password_hash,
			email_verified = :email_verified,
			updated_at = :updated_at
		WHERE id = :id
	`
	if _, err := r.db.NamedExecContext(ctx, query, acc); err != nil {
		return fmt.Errorf("AccountRepository.Update: %w", err)
	}
	return nil
}

func (r *AccountRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE users.accounts SET status = $1, updated_at = NOW() WHERE id = $2`
	if _, err := r.db.ExecContext(ctx, query, status, id); err != nil {
		return fmt.Errorf("AccountRepository.UpdateStatus: %w", err)
	}
	return nil
}

func (r *AccountRepository) UpdateRole(ctx context.Context, id uuid.UUID, role string) error {
	query := `UPDATE users.accounts SET role = $1, updated_at = NOW() WHERE id = $2`
	if _, err := r.db.ExecContext(ctx, query, role, id); err != nil {
		return fmt.Errorf("AccountRepository.UpdateRole: %w", err)
	}
	return nil
}

func (r *AccountRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users.accounts SET last_login_at = NOW() WHERE id = $1`
	if _, err := r.db.ExecContext(ctx, query, id); err != nil {
		return fmt.Errorf("AccountRepository.UpdateLastLogin: %w", err)
	}
	return nil
}

func (r *AccountRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users.accounts SET deleted_at = NOW() WHERE id = $1`
	if _, err := r.db.ExecContext(ctx, query, id); err != nil {
		return fmt.Errorf("AccountRepository.SoftDelete: %w", err)
	}
	return nil
}

func (r *AccountRepository) List(ctx context.Context, offset, limit int) ([]*domain.Account, int, error) {
	var total int
	if err := r.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM users.accounts WHERE deleted_at IS NULL`,
	); err != nil {
		return nil, 0, fmt.Errorf("AccountRepository.List: count: %w", err)
	}

	var accounts []*domain.Account
	query := `SELECT * FROM users.accounts WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	if err := r.db.SelectContext(ctx, &accounts, query, limit, offset); err != nil {
		return nil, 0, fmt.Errorf("AccountRepository.List: select: %w", err)
	}

	return accounts, total, nil
}

// ─── SessionRepository ────────────────────────────────────────────────────────

// SessionRepository implements domain.ISessionRepository.
type SessionRepository struct {
	db *sqlx.DB
}

func NewSessionRepository(db *sqlx.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, s *domain.Session) error {
	query := `
		INSERT INTO users.sessions
			(id, user_id, refresh_token, ip_address, user_agent, expires_at, created_at)
		VALUES
			(:id, :user_id, :refresh_token, :ip_address, :user_agent, :expires_at, :created_at)
	`
	if _, err := r.db.NamedExecContext(ctx, query, s); err != nil {
		return fmt.Errorf("SessionRepository.Create: %w", err)
	}
	return nil
}

func (r *SessionRepository) GetByToken(ctx context.Context, hashedToken string) (*domain.Session, error) {
	var s domain.Session
	query := `SELECT * FROM users.sessions WHERE refresh_token = $1`
	if err := r.db.GetContext(ctx, &s, query, hashedToken); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, fmt.Errorf("SessionRepository.GetByToken: %w", err)
	}
	return &s, nil
}

func (r *SessionRepository) CountActiveByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users.sessions WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()`
	if err := r.db.GetContext(ctx, &count, query, userID); err != nil {
		return 0, fmt.Errorf("SessionRepository.CountActiveByUser: %w", err)
	}
	return count, nil
}

func (r *SessionRepository) RevokeByToken(ctx context.Context, hashedToken string) error {
	query := `UPDATE users.sessions SET revoked_at = NOW() WHERE refresh_token = $1`
	if _, err := r.db.ExecContext(ctx, query, hashedToken); err != nil {
		return fmt.Errorf("SessionRepository.RevokeByToken: %w", err)
	}
	return nil
}

func (r *SessionRepository) RevokeAllByUser(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE users.sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	if _, err := r.db.ExecContext(ctx, query, userID); err != nil {
		return fmt.Errorf("SessionRepository.RevokeAllByUser: %w", err)
	}
	return nil
}

func (r *SessionRepository) DeleteExpired(ctx context.Context) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM users.sessions WHERE expires_at < NOW()`)
	if err != nil {
		return 0, fmt.Errorf("SessionRepository.DeleteExpired: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ─── APIKeyRepository ─────────────────────────────────────────────────────────

// APIKeyRepository implements domain.IAPIKeyRepository.
type APIKeyRepository struct {
	db *sqlx.DB
}

func NewAPIKeyRepository(db *sqlx.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(ctx context.Context, k *domain.APIKey) error {
	query := `
		INSERT INTO users.api_keys
			(id, user_id, name, key_hash, key_prefix, scopes, created_at)
		VALUES
			(:id, :user_id, :name, :key_hash, :key_prefix, :scopes, :created_at)
	`
	if _, err := r.db.NamedExecContext(ctx, query, k); err != nil {
		return fmt.Errorf("APIKeyRepository.Create: %w", err)
	}
	return nil
}

func (r *APIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	var k domain.APIKey
	query := `SELECT * FROM users.api_keys WHERE key_hash = $1 AND revoked_at IS NULL`
	if err := r.db.GetContext(ctx, &k, query, keyHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("APIKeyRepository.GetByHash: %w", err)
	}
	return &k, nil
}

func (r *APIKeyRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.APIKey, error) {
	var keys []*domain.APIKey
	query := `SELECT * FROM users.api_keys WHERE user_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC`
	if err := r.db.SelectContext(ctx, &keys, query, userID); err != nil {
		return nil, fmt.Errorf("APIKeyRepository.ListByUser: %w", err)
	}
	return keys, nil
}

func (r *APIKeyRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	if _, err := r.db.ExecContext(ctx, `UPDATE users.api_keys SET revoked_at = NOW() WHERE id = $1`, id); err != nil {
		return fmt.Errorf("APIKeyRepository.Revoke: %w", err)
	}
	return nil
}

func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	if _, err := r.db.ExecContext(ctx, `UPDATE users.api_keys SET last_used_at = NOW() WHERE id = $1`, id); err != nil {
		return fmt.Errorf("APIKeyRepository.UpdateLastUsed: %w", err)
	}
	return nil
}
