package events

import (
	"context"
	"fmt"

	natspkg "github.com/pvp/pkg/nats"
	"github.com/pvp/proxy-service/internal/domain"
)

const (
	TopicOrderCreated   = "proxy.order.created"
	TopicOrderFulfilled = "proxy.order.fulfilled"
	TopicOrderCancelled = "proxy.order.cancelled"
	TopicOrderFailed    = "proxy.order.failed"
)

// Publisher implements domain.IEventPublisher.
type Publisher struct{ pub *natspkg.Publisher }

func NewPublisher(pub *natspkg.Publisher) *Publisher { return &Publisher{pub: pub} }

func (p *Publisher) PublishOrderCreated(ctx context.Context, order *domain.Order) error {
	if err := p.pub.Publish(ctx, TopicOrderCreated, map[string]any{
		"order_id": order.ID, "user_id": order.UserID, "product_id": order.ProductID,
	}); err != nil {
		return fmt.Errorf("PublishOrderCreated: %w", err)
	}
	return nil
}

func (p *Publisher) PublishOrderFulfilled(ctx context.Context, order *domain.Order) error {
	if err := p.pub.Publish(ctx, TopicOrderFulfilled, map[string]any{
		"order_id": order.ID, "user_id": order.UserID,
	}); err != nil {
		return fmt.Errorf("PublishOrderFulfilled: %w", err)
	}
	return nil
}

func (p *Publisher) PublishOrderCancelled(ctx context.Context, order *domain.Order) error {
	if err := p.pub.Publish(ctx, TopicOrderCancelled, map[string]any{
		"order_id": order.ID, "user_id": order.UserID,
	}); err != nil {
		return fmt.Errorf("PublishOrderCancelled: %w", err)
	}
	return nil
}

func (p *Publisher) PublishOrderFailed(ctx context.Context, orderID interface{}, reason string) error {
	if err := p.pub.Publish(ctx, TopicOrderFailed, map[string]any{
		"order_id": orderID, "reason": reason,
	}); err != nil {
		return fmt.Errorf("PublishOrderFailed: %w", err)
	}
	return nil
}
