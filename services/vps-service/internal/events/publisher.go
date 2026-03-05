package events

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	natspkg "github.com/pvp/pkg/nats"
	"github.com/pvp/vps-service/internal/domain"
	"github.com/shopspring/decimal"
)

const (
	TopicStateChanged        = "vm.state.changed"
	TopicProvisionRequested  = "vm.provision.requested"
	TopicVMReady             = "vm.ready"
	TopicProvisionFailed     = "vm.provision.failed"
	TopicSuspended           = "vm.suspended"
	TopicTerminated          = "vm.terminated"
	TopicUsageReport         = "vps.usage.report"
)

// Publisher implements domain.IEventPublisher.
type Publisher struct{ pub *natspkg.Publisher }

func NewPublisher(pub *natspkg.Publisher) *Publisher { return &Publisher{pub: pub} }

func (p *Publisher) PublishStateChanged(ctx context.Context, instanceID uuid.UUID, from, to string) error {
	return p.pub.Publish(ctx, TopicStateChanged, map[string]any{
		"instance_id": instanceID, "from": from, "to": to,
	})
}

func (p *Publisher) PublishProvisionRequested(ctx context.Context, inst *domain.Instance) error {
	return p.pub.Publish(ctx, TopicProvisionRequested, map[string]any{
		"instance_id": inst.ID, "node_name": inst.NodeName,
		"user_id": inst.UserID, "plan_id": inst.PlanID,
	})
}

func (p *Publisher) PublishVMReady(ctx context.Context, inst *domain.Instance) error {
	return p.pub.Publish(ctx, TopicVMReady, map[string]any{
		"instance_id": inst.ID, "user_id": inst.UserID,
		"ip_address": inst.IPAddress, "hostname": inst.Hostname,
	})
}

func (p *Publisher) PublishProvisionFailed(ctx context.Context, instanceID uuid.UUID, reason, step string) error {
	return p.pub.Publish(ctx, TopicProvisionFailed, map[string]any{
		"instance_id": instanceID, "reason": reason, "step": step,
	})
}

func (p *Publisher) PublishSuspended(ctx context.Context, instanceID uuid.UUID, reason string) error {
	return p.pub.Publish(ctx, TopicSuspended, map[string]any{
		"instance_id": instanceID, "reason": reason,
	})
}

func (p *Publisher) PublishTerminated(ctx context.Context, instanceID, userID uuid.UUID) error {
	return p.pub.Publish(ctx, TopicTerminated, map[string]any{
		"instance_id": instanceID, "user_id": userID,
	})
}

func (p *Publisher) PublishUsageReport(ctx context.Context, instanceID uuid.UUID, hours float64, amount decimal.Decimal) error {
	if err := p.pub.Publish(ctx, TopicUsageReport, map[string]any{
		"instance_id": instanceID, "hours": hours, "amount": amount.String(),
	}); err != nil {
		return fmt.Errorf("PublishUsageReport: %w", err)
	}
	return nil
}
