package events

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	natspkg "github.com/pvp/pkg/nats"
)

// Topics for user-service events.
const (
	TopicUserRegistered      = "user.registered"
	TopicUserVerified        = "user.verified"
	TopicUserLogin           = "user.login"
	TopicPasswordChanged     = "user.password_changed"
	TopicUserSuspended       = "user.suspended"
	// Audit topics
	TopicUser2FAEnabled      = "user.2fa_enabled"
	TopicUser2FADisabled     = "user.2fa_disabled"
	TopicUser2FAAdminDisable = "user.2fa_admin_disabled"
	TopicUserAdminUpdated    = "user.admin_updated"
)

// Publisher implements domain.IEventPublisher.
type Publisher struct {
	pub *natspkg.Publisher
}

// NewPublisher creates a Publisher.
func NewPublisher(pub *natspkg.Publisher) *Publisher {
	return &Publisher{pub: pub}
}

func (p *Publisher) PublishUserRegistered(ctx context.Context, userID uuid.UUID, email string) error {
	payload := map[string]string{"user_id": userID.String(), "email": email}
	if err := p.pub.Publish(ctx, TopicUserRegistered, payload); err != nil {
		return fmt.Errorf("Publisher.PublishUserRegistered: %w", err)
	}
	return nil
}

func (p *Publisher) PublishUserVerified(ctx context.Context, userID uuid.UUID) error {
	payload := map[string]string{"user_id": userID.String()}
	if err := p.pub.Publish(ctx, TopicUserVerified, payload); err != nil {
		return fmt.Errorf("Publisher.PublishUserVerified: %w", err)
	}
	return nil
}

func (p *Publisher) PublishUserLogin(ctx context.Context, userID uuid.UUID, ip string) error {
	payload := map[string]string{"user_id": userID.String(), "ip": ip}
	if err := p.pub.Publish(ctx, TopicUserLogin, payload); err != nil {
		return fmt.Errorf("Publisher.PublishUserLogin: %w", err)
	}
	return nil
}

func (p *Publisher) PublishPasswordChanged(ctx context.Context, userID uuid.UUID) error {
	payload := map[string]string{"user_id": userID.String()}
	if err := p.pub.Publish(ctx, TopicPasswordChanged, payload); err != nil {
		return fmt.Errorf("Publisher.PublishPasswordChanged: %w", err)
	}
	return nil
}

func (p *Publisher) PublishUserSuspended(ctx context.Context, userID uuid.UUID, reason string) error {
	payload := map[string]any{"user_id": userID.String(), "reason": reason}
	if err := p.pub.Publish(ctx, TopicUserSuspended, payload); err != nil {
		return fmt.Errorf("Publisher.PublishUserSuspended: %w", err)
	}
	return nil
}

func (p *Publisher) PublishUser2FAEnabled(ctx context.Context, userID uuid.UUID) error {
	payload := map[string]any{"user_id": userID.String()}
	if err := p.pub.Publish(ctx, TopicUser2FAEnabled, payload); err != nil {
		return fmt.Errorf("Publisher.PublishUser2FAEnabled: %w", err)
	}
	return nil
}

func (p *Publisher) PublishUser2FADisabled(ctx context.Context, userID uuid.UUID) error {
	payload := map[string]any{"user_id": userID.String()}
	if err := p.pub.Publish(ctx, TopicUser2FADisabled, payload); err != nil {
		return fmt.Errorf("Publisher.PublishUser2FADisabled: %w", err)
	}
	return nil
}

func (p *Publisher) PublishUser2FAAdminDisabled(ctx context.Context, userID, actorID uuid.UUID) error {
	payload := map[string]any{"user_id": userID.String(), "actor_id": actorID.String()}
	if err := p.pub.Publish(ctx, TopicUser2FAAdminDisable, payload); err != nil {
		return fmt.Errorf("Publisher.PublishUser2FAAdminDisabled: %w", err)
	}
	return nil
}

func (p *Publisher) PublishUserAdminUpdated(ctx context.Context, userID, actorID uuid.UUID, changes map[string]any) error {
	payload := map[string]any{"user_id": userID.String(), "actor_id": actorID.String(), "changes": changes}
	if err := p.pub.Publish(ctx, TopicUserAdminUpdated, payload); err != nil {
		return fmt.Errorf("Publisher.PublishUserAdminUpdated: %w", err)
	}
	return nil
}
