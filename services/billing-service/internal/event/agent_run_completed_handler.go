package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/open-mrp/api/services/billing-service/internal/domain"
	"github.com/open-mrp/api/services/billing-service/internal/service"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type AgentTokenBillingHandler struct {
	tokenBillingRepo domain.AgentTokenBillingRepo
	repoFactory      domain.RepoFactory
	txManager        service.TransactionManager
	tracer           trace.Tracer
}

func NewAgentTokenBillingHandler(
	tokenBillingRepo domain.AgentTokenBillingRepo,
	repoFactory domain.RepoFactory,
	txManager service.TransactionManager,
) *AgentTokenBillingHandler {
	return &AgentTokenBillingHandler{
		tokenBillingRepo: tokenBillingRepo,
		repoFactory:      repoFactory,
		txManager:        txManager,
		tracer:           tracing.GetTracer("billing-service.agent_token_billing_handler"),
	}
}

func (h *AgentTokenBillingHandler) Handle(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := h.tracer.Start(ctx, "handler.agent_token_billing",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.routing_key", msg.RoutingKey),
			attribute.String("messaging.message_id", msg.MessageId),
		),
	)
	defer span.End()

	var amqpMsg contracts.AmqpMessage
	if err := json.Unmarshal(msg.Body, &amqpMsg); err != nil {
		log.Printf("[agent_token_billing] Failed to unmarshal AMQP message: %v", err)
		span.RecordError(err)
		return err
	}

	var data messaging.AgentRunCompletedData
	if err := json.Unmarshal(amqpMsg.Data, &data); err != nil {
		log.Printf("[agent_token_billing] Failed to unmarshal run completed data: %v", err)
		span.RecordError(err)
		return err
	}

	// Use the billing account ID for all billing operations. For sandbox accounts, this is the production owner's account ID. Falls back to AccountID for backwards compatibility with events published before this field existed.
	billingAcctID := data.BillingAccountID
	if billingAcctID == "" {
		billingAcctID = data.AccountID
	}

	span.SetAttributes(
		attribute.String("agent.run_id", data.AgentRunID),
		attribute.String("agent.account_id", data.AccountID),
		attribute.String("agent.billing_account_id", billingAcctID),
		attribute.Int("agent.total_tokens", data.TotalTokens),
	)

	usageRepo := h.repoFactory.NewAccountUsageRepo()

	subInfo, apiErr := usageRepo.GetAccountSubscriptionInfo(ctx, billingAcctID)
	if apiErr != nil {
		return fmt.Errorf("failed to get subscription info: %w", apiErr)
	}

	var periodStart, periodEnd time.Time
	if subInfo.SubscriptionCurrentPeriodEnd != nil {
		periodEnd = *subInfo.SubscriptionCurrentPeriodEnd
		periodStart = periodEnd.AddDate(0, -1, 0)
	} else {
		now := time.Now().UTC()
		periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		periodEnd = periodStart.AddDate(0, 1, 0)
	}

	billingID, genErr := id.GenID(id.AgentTokenUsageIDPrefix, nil)
	if genErr != nil {
		return fmt.Errorf("failed to generate billing ID: %w", genErr)
	}

	// The upsert accumulates (total_tokens = total_tokens + VALUES(...)), so unlike the convergent handlers a second delivery does not land on the same total — it inflates it. The recovery point commits with the row so the delivery cannot be applied twice; without it a process killed between this commit and the marker leaves a message that looks unprocessed and is counted again.
	apiErr = h.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		if apiErr := f.NewAgentTokenBillingRepo().UpsertAgentTokenBilling(txCtx, domain.UpsertAgentTokenBillingParams{
			ID:           billingID,
			AccountID:    billingAcctID,
			PeriodStart:  periodStart,
			PeriodEnd:    periodEnd,
			InputTokens:  int64(data.InputTokens),
			OutputTokens: int64(data.OutputTokens),
			TotalTokens:  int64(data.TotalTokens),
		}); apiErr != nil {
			return apiErr
		}
		return completeInboxRecord(txCtx, f)
	})
	if apiErr != nil {
		return fmt.Errorf("failed to upsert agent token billing: %w", apiErr)
	}

	log.Printf("[agent_token_billing] Recorded %d tokens for account %s (analytics only, Stripe metering via gateway)",
		data.TotalTokens, billingAcctID)

	return nil
}
