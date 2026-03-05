package events

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	natspkg "github.com/pvp/pkg/nats"
	"github.com/pvp/reseller-service/internal/domain"
	"github.com/shopspring/decimal"
)

const (
	TopicResellerCreated       = "reseller.created"
	TopicResellerApproved      = "reseller.approved"
	TopicResellerSuspended     = "reseller.suspended"
	TopicPricingUpdated        = "reseller.pricing.updated"
)

// Publisher implements domain.IEventPublisher.
type Publisher struct{ pub *natspkg.Publisher }

func NewPublisher(pub *natspkg.Publisher) *Publisher { return &Publisher{pub: pub} }

func (p *Publisher) PublishCreated(ctx context.Context, r *domain.ResellerAccount) error {
	if err := p.pub.Publish(ctx, TopicResellerCreated, map[string]any{
		"reseller_id": r.ID, "user_id": r.UserID, "company": r.CompanyName,
	}); err != nil {
		return fmt.Errorf("PublishCreated: %w", err)
	}
	return nil
}

func (p *Publisher) PublishApproved(ctx context.Context, resellerID uuid.UUID) error {
	return p.pub.Publish(ctx, TopicResellerApproved, map[string]any{"reseller_id": resellerID})
}

func (p *Publisher) PublishSuspended(ctx context.Context, resellerID uuid.UUID, reason string) error {
	return p.pub.Publish(ctx, TopicResellerSuspended, map[string]any{
		"reseller_id": resellerID, "reason": reason,
	})
}

func (p *Publisher) PublishPricingUpdated(ctx context.Context, resellerID, productID uuid.UUID, newPrice decimal.Decimal) error {
	return p.pub.Publish(ctx, TopicPricingUpdated, map[string]any{
		"reseller_id": resellerID, "product_id": productID, "new_price": newPrice.String(),
	})
}
