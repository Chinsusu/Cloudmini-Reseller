// Package usecase contains all business logic for user-service.
// auth_usecase.go handles: Register, Login, RefreshToken, Logout,
// ForgotPassword, ResetPassword, VerifyEmail.
package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"regexp"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pvp/user-service/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// AuthUsecase handles authentication flows.
type AuthUsecase struct {
	accountRepo domain.IAccountRepository
	sessionRepo domain.ISessionRepository
	eventPub    domain.IEventPublisher
	logger      *slog.Logger

	jwtSecret          []byte
	jwtAccessTTL       time.Duration
	jwtRefreshTTL      time.Duration
	adminRefreshTTL    time.Duration
	maxSessions        int
	maxLoginAttempts   int
}

// NewAuthUsecase constructs AuthUsecase via dependency injection.
func NewAuthUsecase(
	accountRepo domain.IAccountRepository,
	sessionRepo domain.ISessionRepository,
	eventPub domain.IEventPublisher,
	logger *slog.Logger,
	jwtSecret []byte,
	jwtAccessTTL, jwtRefreshTTL, adminRefreshTTL time.Duration,
	maxSessions, maxLoginAttempts int,
) *AuthUsecase {
	return &AuthUsecase{
		accountRepo:      accountRepo,
		sessionRepo:      sessionRepo,
		eventPub:         eventPub,
		logger:           logger,
		jwtSecret:        jwtSecret,
		jwtAccessTTL:     jwtAccessTTL,
		jwtRefreshTTL:    jwtRefreshTTL,
		adminRefreshTTL:  adminRefreshTTL,
		maxSessions:      maxSessions,
		maxLoginAttempts: maxLoginAttempts,
	}
}

// RegisterRequest is the input for Register.
type RegisterRequest struct {
	Email    string
	Password string
	FullName string
	Phone    string
}

// RegisterResult is the output of Register.
type RegisterResult struct {
	UserID string
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Register creates a new user account.
// Returns RegisterResult with user ID on success.
// Does NOT return a JWT — user must verify email first.
func (u *AuthUsecase) Register(ctx context.Context, req RegisterRequest) (*RegisterResult, error) {
	// Validate email
	if !emailRegex.MatchString(req.Email) {
		return nil, domain.ErrWeakPassword
	}

	// Validate password strength
	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}

	// Check email uniqueness
	existing, err := u.accountRepo.GetByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return nil, domain.ErrEmailAlreadyExists
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("AuthUsecase.Register: hash password: %w", err)
	}

	acc := &domain.Account{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: string(hash),
		FullName:     req.FullName,
		Phone:        req.Phone,
		Role:         domain.RoleUser,
		Status:       domain.StatusPendingVerification,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := u.accountRepo.Create(ctx, acc); err != nil {
		return nil, fmt.Errorf("AuthUsecase.Register: create account: %w", err)
	}

	// Publish event asynchronously (fire and forget, logged separately)
	go func() {
		if pubErr := u.eventPub.PublishUserRegistered(context.Background(), acc.ID, acc.Email); pubErr != nil {
			u.logger.ErrorContext(ctx, "failed to publish user.registered",
				slog.String("user_id", acc.ID.String()),
				slog.String("error", pubErr.Error()),
			)
		}
	}()

	u.logger.InfoContext(ctx, "user registered",
		slog.String("user_id", acc.ID.String()),
		slog.String("email", acc.Email),
	)

	return &RegisterResult{UserID: acc.ID.String()}, nil
}

// LoginRequest is the input for Login.
type LoginRequest struct {
	Email     string
	Password  string
	IPAddress string
	UserAgent string
}

// LoginResult contains the JWT tokens and user info.
type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	UserID       string
	Role         string
}

// Login authenticates a user and issues JWT + refresh token.
func (u *AuthUsecase) Login(ctx context.Context, req LoginRequest) (*LoginResult, error) {
	acc, err := u.accountRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// Check account status
	if acc.Status == domain.StatusSuspended || acc.Status == domain.StatusBanned {
		return nil, domain.ErrAccountSuspended
	}
	if !acc.EmailVerified && acc.Status == domain.StatusPendingVerification {
		return nil, domain.ErrEmailNotVerified
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(req.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// Check max sessions
	count, err := u.sessionRepo.CountActiveByUser(ctx, acc.ID)
	if err != nil {
		return nil, fmt.Errorf("AuthUsecase.Login: count sessions: %w", err)
	}
	if count >= u.maxSessions {
		// Evict oldest by revoking all and creating fresh — simplest policy
		if err := u.sessionRepo.RevokeAllByUser(ctx, acc.ID); err != nil {
			return nil, fmt.Errorf("AuthUsecase.Login: evict sessions: %w", err)
		}
	}

	// Issue tokens
	refreshTTL := u.jwtRefreshTTL
	if acc.Role == domain.RoleAdmin || acc.Role == domain.RoleSuperAdmin {
		refreshTTL = u.adminRefreshTTL
	}

	accessToken, err := u.issueAccessToken(acc)
	if err != nil {
		return nil, fmt.Errorf("AuthUsecase.Login: issue access token: %w", err)
	}

	rawRefresh, hashedRefresh := generateRefreshToken()
	session := &domain.Session{
		ID:           uuid.New(),
		UserID:       acc.ID,
		RefreshToken: hashedRefresh,
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		ExpiresAt:    time.Now().Add(refreshTTL),
		CreatedAt:    time.Now(),
	}
	if err := u.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("AuthUsecase.Login: create session: %w", err)
	}

	if err := u.accountRepo.UpdateLastLogin(ctx, acc.ID); err != nil {
		u.logger.WarnContext(ctx, "failed to update last_login_at", slog.String("user_id", acc.ID.String()))
	}

	go func() {
		_ = u.eventPub.PublishUserLogin(context.Background(), acc.ID, req.IPAddress)
	}()

	u.logger.InfoContext(ctx, "user logged in",
		slog.String("user_id", acc.ID.String()),
		slog.String("ip", req.IPAddress),
	)

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresAt:    time.Now().Add(u.jwtAccessTTL),
		UserID:       acc.ID.String(),
		Role:         acc.Role,
	}, nil
}

// RefreshTokenRequest contains the raw refresh token.
type RefreshTokenRequest struct {
	RefreshToken string
	IPAddress    string
	UserAgent    string
}

// RefreshToken rotates the refresh token and issues a new access token.
func (u *AuthUsecase) RefreshToken(ctx context.Context, req RefreshTokenRequest) (*LoginResult, error) {
	hashed := hashToken(req.RefreshToken)

	session, err := u.sessionRepo.GetByToken(ctx, hashed)
	if err != nil {
		return nil, domain.ErrSessionNotFound
	}
	if session.RevokedAt != nil {
		return nil, domain.ErrSessionRevoked
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, domain.ErrTokenExpired
	}

	acc, err := u.accountRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	// Rotate: revoke old, issue new
	if err := u.sessionRepo.RevokeByToken(ctx, hashed); err != nil {
		return nil, fmt.Errorf("AuthUsecase.RefreshToken: revoke: %w", err)
	}

	accessToken, err := u.issueAccessToken(acc)
	if err != nil {
		return nil, fmt.Errorf("AuthUsecase.RefreshToken: issue access token: %w", err)
	}

	refreshTTL := u.jwtRefreshTTL
	if acc.Role == domain.RoleAdmin || acc.Role == domain.RoleSuperAdmin {
		refreshTTL = u.adminRefreshTTL
	}

	rawRefresh, hashedRefresh := generateRefreshToken()
	newSession := &domain.Session{
		ID:           uuid.New(),
		UserID:       acc.ID,
		RefreshToken: hashedRefresh,
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		ExpiresAt:    time.Now().Add(refreshTTL),
		CreatedAt:    time.Now(),
	}
	if err := u.sessionRepo.Create(ctx, newSession); err != nil {
		return nil, fmt.Errorf("AuthUsecase.RefreshToken: create session: %w", err)
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresAt:    time.Now().Add(u.jwtAccessTTL),
		UserID:       acc.ID.String(),
		Role:         acc.Role,
	}, nil
}

// Logout revokes the provided refresh token.
func (u *AuthUsecase) Logout(ctx context.Context, rawRefreshToken string) error {
	hashed := hashToken(rawRefreshToken)
	if err := u.sessionRepo.RevokeByToken(ctx, hashed); err != nil {
		return fmt.Errorf("AuthUsecase.Logout: %w", err)
	}
	return nil
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

type jwtClaims struct {
	UserID     string `json:"sub"`
	Role       string `json:"role"`
	ResellerID string `json:"reseller_id,omitempty"`
	Email      string `json:"email"`
	jwt.RegisteredClaims
}

func (u *AuthUsecase) issueAccessToken(acc *domain.Account) (string, error) {
	resellerID := ""
	if acc.ResellerID != nil {
		resellerID = acc.ResellerID.String()
	}

	claims := jwtClaims{
		UserID:     acc.ID.String(),
		Role:       acc.Role,
		ResellerID: resellerID,
		Email:      acc.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(u.jwtAccessTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(u.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("issueAccessToken: sign: %w", err)
	}
	return signed, nil
}

// generateRefreshToken returns (rawToken, sha256Hash).
func generateRefreshToken() (raw, hashed string) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	raw = hex.EncodeToString(b)
	return raw, hashToken(raw)
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return domain.ErrWeakPassword
	}
	var hasUpper, hasLower, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return domain.ErrWeakPassword
	}
	return nil
}
