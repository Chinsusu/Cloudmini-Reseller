package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/pvp/user-service/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

// UserUsecase handles profile management.
type UserUsecase struct {
	accountRepo domain.IAccountRepository
	eventPub    domain.IEventPublisher
	logger      *slog.Logger
}

// NewUserUsecase constructs UserUsecase.
func NewUserUsecase(accountRepo domain.IAccountRepository, eventPub domain.IEventPublisher, logger *slog.Logger) *UserUsecase {
	return &UserUsecase{accountRepo: accountRepo, eventPub: eventPub, logger: logger}
}

// GetProfile returns a user's account details.
func (u *UserUsecase) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.Account, error) {
	acc, err := u.accountRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("UserUsecase.GetProfile: %w", err)
	}
	return acc, nil
}

// UpdateProfileRequest is the input for UpdateProfile.
type UpdateProfileRequest struct {
	FullName string
	Phone    string
}

// UpdateProfile updates the user's public profile.
func (u *UserUsecase) UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*domain.Account, error) {
	acc, err := u.accountRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("UserUsecase.UpdateProfile: get: %w", err)
	}

	acc.FullName = req.FullName
	acc.Phone = req.Phone
	acc.UpdatedAt = time.Now()

	if err := u.accountRepo.Update(ctx, acc); err != nil {
		return nil, fmt.Errorf("UserUsecase.UpdateProfile: update: %w", err)
	}

	u.logger.InfoContext(ctx, "user profile updated", slog.String("user_id", userID.String()))
	return acc, nil
}

// ChangePassword validates the old password and sets a new one.
func (u *UserUsecase) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	acc, err := u.accountRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("UserUsecase.ChangePassword: get: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(oldPassword)); err != nil {
		return domain.ErrInvalidCredentials
	}

	if err := validatePassword(newPassword); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("UserUsecase.ChangePassword: hash: %w", err)
	}

	acc.PasswordHash = string(hash)
	acc.UpdatedAt = time.Now()
	if err := u.accountRepo.Update(ctx, acc); err != nil {
		return fmt.Errorf("UserUsecase.ChangePassword: update: %w", err)
	}

	u.logger.InfoContext(ctx, "user password changed", slog.String("user_id", userID.String()))
	return nil
}

// UpdateStatus (admin) changes a user's status.
func (u *UserUsecase) UpdateStatus(ctx context.Context, adminID, targetUserID uuid.UUID, status string) error {
	switch status {
	case domain.StatusActive, domain.StatusSuspended, domain.StatusBanned:
		// valid
	default:
		return fmt.Errorf("UserUsecase.UpdateStatus: invalid status %q", status)
	}

	if err := u.accountRepo.UpdateStatus(ctx, targetUserID, status); err != nil {
		return fmt.Errorf("UserUsecase.UpdateStatus: %w", err)
	}

	u.logger.InfoContext(ctx, "user status updated",
		slog.String("admin_id", adminID.String()),
		slog.String("user_id", targetUserID.String()),
		slog.String("status", status),
	)
	return nil
}

// UpdateRole (admin) changes a user's role.
func (u *UserUsecase) UpdateRole(ctx context.Context, adminID, targetUserID uuid.UUID, role string) error {
	switch role {
	case domain.RoleUser, domain.RoleReseller, domain.RoleAdmin:
		// valid (super_admin only set manually)
	default:
		return fmt.Errorf("UserUsecase.UpdateRole: invalid role %q", role)
	}

	if err := u.accountRepo.UpdateRole(ctx, targetUserID, role); err != nil {
		return fmt.Errorf("UserUsecase.UpdateRole: %w", err)
	}

	u.logger.InfoContext(ctx, "user role updated",
		slog.String("admin_id", adminID.String()),
		slog.String("user_id", targetUserID.String()),
		slog.String("role", role),
	)
	return nil
}

// ListUsers returns paginated user accounts (admin).
func (u *UserUsecase) ListUsers(ctx context.Context, offset, limit int) ([]*domain.Account, int, error) {
	users, total, err := u.accountRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("UserUsecase.ListUsers: %w", err)
	}
	return users, total, nil
}

// SoftDelete marks a user account as deleted (admin).
func (u *UserUsecase) SoftDelete(ctx context.Context, userID uuid.UUID) error {
	if err := u.accountRepo.SoftDelete(ctx, userID); err != nil {
		return fmt.Errorf("UserUsecase.SoftDelete: %w", err)
	}
	u.logger.InfoContext(ctx, "user soft-deleted", slog.String("user_id", userID.String()))
	return nil
}

// SetupTOTPResult holds the secret and provisioning URI for QR code generation.
type SetupTOTPResult struct {
	Secret     string `json:"secret"`
	OtpauthURL string `json:"otpauth_url"`
}

// SetupTOTP generates a new TOTP secret for the user (not enabled yet).
func (u *UserUsecase) SetupTOTP(ctx context.Context, userID uuid.UUID) (*SetupTOTPResult, error) {
	acc, err := u.accountRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("UserUsecase.SetupTOTP: %w", err)
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Cloudmini",
		AccountName: acc.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("UserUsecase.SetupTOTP: generate: %w", err)
	}
	// Store secret (not enabled yet)
	secret := key.Secret()
	if err := u.accountRepo.UpdateTOTP(ctx, userID, false, &secret); err != nil {
		return nil, fmt.Errorf("UserUsecase.SetupTOTP: save: %w", err)
	}
	return &SetupTOTPResult{Secret: secret, OtpauthURL: key.URL()}, nil
}

// EnableTOTP verifies the TOTP code and enables 2FA.
func (u *UserUsecase) EnableTOTP(ctx context.Context, userID uuid.UUID, code string) error {
	acc, err := u.accountRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("UserUsecase.EnableTOTP: %w", err)
	}
	if acc.TotpSecret == nil {
		return fmt.Errorf("UserUsecase.EnableTOTP: 2FA not set up yet")
	}
	if !totp.Validate(code, *acc.TotpSecret) {
		return domain.ErrInvalidTOTPCode
	}
	if err := u.accountRepo.UpdateTOTP(ctx, userID, true, acc.TotpSecret); err != nil {
		return err
	}
	go func() { _ = u.eventPub.PublishUser2FAEnabled(context.Background(), userID) }()
	return nil
}

// DisableTOTP verifies the TOTP code and disables 2FA.
func (u *UserUsecase) DisableTOTP(ctx context.Context, userID uuid.UUID, code string) error {
	acc, err := u.accountRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("UserUsecase.DisableTOTP: %w", err)
	}
	if !acc.TotpEnabled {
		return nil // already disabled
	}
	if !totp.Validate(code, *acc.TotpSecret) {
		return domain.ErrInvalidTOTPCode
	}
	if err := u.accountRepo.UpdateTOTP(ctx, userID, false, nil); err != nil {
		return err
	}
	go func() { _ = u.eventPub.PublishUser2FADisabled(context.Background(), userID) }()
	return nil
}

// AdminDisableTOTP force-disables 2FA for any user (no code needed).
func (u *UserUsecase) AdminDisableTOTP(ctx context.Context, userID, actorID uuid.UUID) error {
	if err := u.accountRepo.UpdateTOTP(ctx, userID, false, nil); err != nil {
		return err
	}
	go func() { _ = u.eventPub.PublishUser2FAAdminDisabled(context.Background(), userID, actorID) }()
	return nil
}

// ─── APIKeyUsecase ─────────────────────────────────────────────────────────── 

// APIKeyUsecase manages API key lifecycle.
type APIKeyUsecase struct {
	keyRepo domain.IAPIKeyRepository
	logger  *slog.Logger
}

// NewAPIKeyUsecase constructs APIKeyUsecase.
func NewAPIKeyUsecase(keyRepo domain.IAPIKeyRepository, logger *slog.Logger) *APIKeyUsecase {
	return &APIKeyUsecase{keyRepo: keyRepo, logger: logger}
}

// CreateAPIKeyRequest is the input for creating a new API key.
type CreateAPIKeyRequest struct {
	UserID uuid.UUID
	Name   string
	Scopes []string
}

// CreateAPIKeyResult contains the raw key (shown once) and key metadata.
type CreateAPIKeyResult struct {
	RawKey string
	APIKey *domain.APIKey
}

// CreateAPIKey generates a new API key for a user.
// The raw key is returned once and never stored — only the SHA256 hash is stored.
func (u *APIKeyUsecase) CreateAPIKey(ctx context.Context, req CreateAPIKeyRequest) (*CreateAPIKeyResult, error) {
	rawKey, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("APIKeyUsecase.CreateAPIKey: generate: %w", err)
	}

	hash := hashToken(rawKey)
	prefix := rawKey[:8]

	key := &domain.APIKey{
		ID:        uuid.New(),
		UserID:    req.UserID,
		Name:      req.Name,
		KeyHash:   hash,
		KeyPrefix: prefix,
		Scopes:    req.Scopes,
		CreatedAt: time.Now(),
	}

	if err := u.keyRepo.Create(ctx, key); err != nil {
		return nil, fmt.Errorf("APIKeyUsecase.CreateAPIKey: save: %w", err)
	}

	u.logger.InfoContext(ctx, "api key created",
		slog.String("user_id", req.UserID.String()),
		slog.String("key_id", key.ID.String()),
		slog.String("name", key.Name),
	)

	return &CreateAPIKeyResult{RawKey: rawKey, APIKey: key}, nil
}

// ListAPIKeys returns all non-revoked API keys for a user.
func (u *APIKeyUsecase) ListAPIKeys(ctx context.Context, userID uuid.UUID) ([]*domain.APIKey, error) {
	keys, err := u.keyRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("APIKeyUsecase.ListAPIKeys: %w", err)
	}
	return keys, nil
}

// RevokeAPIKey marks an API key as revoked.
func (u *APIKeyUsecase) RevokeAPIKey(ctx context.Context, userID, keyID uuid.UUID) error {
	// Fetch to confirm ownership
	keys, err := u.keyRepo.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("APIKeyUsecase.RevokeAPIKey: list: %w", err)
	}
	found := false
	for _, k := range keys {
		if k.ID == keyID {
			found = true
			break
		}
	}
	if !found {
		return domain.ErrAPIKeyNotFound
	}

	if err := u.keyRepo.Revoke(ctx, keyID); err != nil {
		return fmt.Errorf("APIKeyUsecase.RevokeAPIKey: revoke: %w", err)
	}

	u.logger.InfoContext(ctx, "api key revoked",
		slog.String("user_id", userID.String()),
		slog.String("key_id", keyID.String()),
	)
	return nil
}

func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "pvp_" + hex.EncodeToString(b), nil
}
