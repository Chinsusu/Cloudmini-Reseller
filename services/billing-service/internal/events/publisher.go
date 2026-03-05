package events

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	natspkg "github.com/pvp/pkg/nats"
	"github.com/shopspring/decimal"
)

const (
	TopicBillingCharged         = "billing.charged"
	TopicBillingDepositCompleted = "billing.deposit.completed"
	TopicBillingWalletLow       = "billing.wallet.low"
	TopicBillingWalletEmpty     = "billing.wallet.empty"
)

// Publisher implements domain.IEventPublisher.
type Publisher struct{ pub *natspkg.Publisher }

func NewPublisher(pub *natspkg.Publisher) *Publisher { return &Publisher{pub: pub} }

func (p *Publisher) PublishCharged(ctx context.Context, userID uuid.UUID, amount decimal.Decimal, ref string) error {
	payload := map[string]string{"user_id": userID.String(), "amount": amount.String(), "reference_type": ref}
	if err := p.pub.Publish(ctx, TopicBillingCharged, payload); err != nil {
		return fmt.Errorf("PublishCharged: %w", err)
	}
	return nil
}

func (p *Publisher) PublishDepositCompleted(ctx context.Context, userID uuid.UUID, amount decimal.Decimal) error {
	payload := map[string]string{"user_id": userID.String(), "amount": amount.String()}
	if err := p.pub.Publish(ctx, TopicBillingDepositCompleted, payload); err != nil {
		return fmt.Errorf("PublishDepositCompleted: %w", err)
	}
	return nil
}

func (p *Publisher) PublishWalletLow(ctx context.Context, userID uuid.UUID, balance decimal.Decimal) error {
	payload := map[string]string{"user_id": userID.String(), "balance": balance.String()}
	if err := p.pub.Publish(ctx, TopicBillingWalletLow, payload); err != nil {
		return fmt.Errorf("PublishWalletLow: %w", err)
	}
	return nil
}

func (p *Publisher) PublishWalletEmpty(ctx context.Context, userID uuid.UUID) error {
	payload := map[string]string{"user_id": userID.String()}
	if err := p.pub.Publish(ctx, TopicBillingWalletEmpty, payload); err != nil {
		return fmt.Errorf("PublishWalletEmpty: %w", err)
	}
	return nil
}
