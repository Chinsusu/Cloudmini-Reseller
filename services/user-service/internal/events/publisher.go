package events

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	natspkg "github.com/pvp/pkg/nats"
)

// Topics for user-service events.
const (
	TopicUserRegistered     = "user.registered"
	TopicUserVerified       = "user.verified"
	TopicUserLogin          = "user.login"
	TopicPasswordChanged    = "user.password_changed"
	TopicUserSuspended      = "user.suspended"
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
