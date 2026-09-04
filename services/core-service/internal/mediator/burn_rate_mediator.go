package mediator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

const burnRateDenominatorUnitID = "day"

var burnRateMedTracer = tracing.GetTracer("core-service.burn_rate_mediator")

// burnRateTimeSpanDays returns elapsed days between two consumption log timestamps. When both fall in the same instant, use one day so multiple same-day events still yield a rate.
func burnRateTimeSpanDays(first, last time.Time) float64 {
	span := last.Sub(first).Hours() / 24
	if span <= 0 {
		return 1
	}
	return span
}

type burnRateMedImpl struct {
	repos domain.RepoFactory
}

type BurnRateMedConfig struct {
	// Repos (required) is the repository factory for burn-rate calculations.
	Repos domain.RepoFactory
}

func (c *BurnRateMedConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("burn rate mediator: repos is required")
	}
	return nil
}

func NewBurnRateMed(config *BurnRateMedConfig) domain.BurnRateMed {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &burnRateMedImpl{repos: config.Repos}
}

// RecalculateFromHistory updates the item's burn_rate from consumption change logs over the last 30 days. When there is insufficient history to compute a new rate the existing value is kept, but the burn rate is still marked fresh.
//
//  1. Load the item and resolve its category's base unit.
//  2. List the item's consumption change logs; keep the existing rate when fewer than two exist.
//  3. Sum the absolute consumption quantities, converting each to the base unit.
//  4. Divide the total by the days elapsed between the first and last log.
//  5. Persist the resulting per-day rate to the item's burn rate.
//
// Every path marks the burn rate fresh (advances rate.updated_at). ListStaleBurnRateItems selects
// items by rate.updated_at, so an item whose recompute yields no new value must still be touched;
// otherwise a genuinely idle item never leaves the stale set and the sweep re-enqueues it forever.
func (m *burnRateMedImpl) RecalculateFromHistory(ctx context.Context, accountID, itemID string) *apierror.APIError {
	ctx, span := burnRateMedTracer.Start(ctx, "mediator.burn_rate.recalculate_from_history")
	defer span.End()
	span.SetAttributes(
		attribute.String("account.id", accountID),
		attribute.String("item.id", itemID),
	)

	itemRepo := m.repos.NewItemRepo()
	item, apiErr := itemRepo.Get(ctx, domain.GetItemParams{
		AccountID: accountID,
		ItemID:    itemID,
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	baseUnitID, _, apiErr := itemRepo.GetCategoryBaseUnitID(ctx, item.ItemCategoryID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	logs, apiErr := itemRepo.ListConsumptionChangeLogsForBurnRate(ctx, accountID, itemID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if len(logs) < 2 {
		return tracing.Trace(span, m.markBurnRateFresh(ctx, item.BurnRateID))
	}

	unitConvRepo := m.repos.NewUnitConversionRepo()
	var totalConsumption decimal.Decimal
	for _, log := range logs {
		measure, err := decimal.NewFromString(log.Value)
		if err != nil {
			return tracing.Trace(span, apierror.NewInternalError(err, "Invalid consumption quantity in change log."))
		}
		absMeasure := measure.Abs()
		normalized, convErr := unitConvRepo.ConvertValue(ctx, absMeasure, log.UnitID, baseUnitID)
		if convErr != nil {
			return tracing.Trace(span, convErr)
		}
		totalConsumption = totalConsumption.Add(normalized)
	}

	if totalConsumption.IsZero() {
		return tracing.Trace(span, m.markBurnRateFresh(ctx, item.BurnRateID))
	}

	timeSpanDays := burnRateTimeSpanDays(logs[0].CreatedAt, logs[len(logs)-1].CreatedAt)
	if timeSpanDays <= 0 {
		return tracing.Trace(span, m.markBurnRateFresh(ctx, item.BurnRateID))
	}

	burnRateMeasure := totalConsumption.Div(decimal.NewFromFloat(timeSpanDays))
	valueStr := burnRateMeasure.String()
	span.SetAttributes(
		attribute.String("rate.id", item.BurnRateID),
		attribute.String("burn_rate.value", valueStr),
	)

	rateRepo := m.repos.NewRateRepo()
	value := valueStr
	num := baseUnitID
	den := burnRateDenominatorUnitID
	_, apiErr = rateRepo.Update(ctx, domain.UpdateRateParams{
		RateID:            item.BurnRateID,
		Value:             &value,
		NumeratorUnitID:   &num,
		DenominatorUnitID: &den,
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

// markBurnRateFresh advances the burn rate's updated_at without changing its value, recording that
// the item was recomputed. Passing only the RateID relies on UpdateRateByID's COALESCE to leave the
// value and units untouched while still setting updated_at = NOW(3), so an item with no new rate to
// write still drops out of the stale-item sweep's window.
func (m *burnRateMedImpl) markBurnRateFresh(ctx context.Context, rateID string) *apierror.APIError {
	_, apiErr := m.repos.NewRateRepo().Update(ctx, domain.UpdateRateParams{RateID: rateID})
	return apiErr
}

// MaybeRecalculateAfterConsumption enqueues a burn-rate recalculation when a consumption change log was recorded. The recompute runs off the caller's transaction via the outbox, so the shared rate row's lock is not held for the length of that transaction.
func MaybeRecalculateAfterConsumption(
	ctx context.Context,
	repos domain.RepoFactory,
	accountID, itemID string,
	delta decimal.Decimal,
	actionType string,
) *apierror.APIError {
	if delta.GreaterThanOrEqual(decimal.Zero) {
		return nil
	}
	// Consumption is booked as 'scan' (production draw-down) for materials/parts and 'system_action'
	// (order fulfillment) for products. 'user_correction' is excluded: manual re-baselines of on-hand
	// counts are not demand and would skew the rate. Keep this set in sync with
	// ListConsumptionChangeLogsForBurnRate's action_type_code filter.
	if actionType != "scan" && actionType != "system_action" {
		return nil
	}
	ctx, span := burnRateMedTracer.Start(ctx, "mediator.burn_rate.maybe_recalculate_after_consumption")
	defer span.End()
	span.SetAttributes(
		attribute.String("account.id", accountID),
		attribute.String("item.id", itemID),
	)
	if apiErr := enqueueBurnRateRecalc(ctx, repos.NewOutboxRepo(), accountID, itemID); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// EnqueueRecalc writes an outbox command to recompute an item's burn rate. Used by the periodic
// sweeper to refresh items no ongoing consumption has touched; the recompute runs on the shared
// consumer, one short transaction per item, so a swept batch never contends as a single unit.
func EnqueueRecalc(ctx context.Context, repos domain.RepoFactory, accountID, itemID string) *apierror.APIError {
	return enqueueBurnRateRecalc(ctx, repos.NewOutboxRepo(), accountID, itemID)
}

// enqueueBurnRateRecalc writes an outbox command to recompute the item's burn rate off the current transaction.
func enqueueBurnRateRecalc(ctx context.Context, outboxRepo messaging.OutboxRepo, accountID, itemID string) *apierror.APIError {
	payload, err := json.Marshal(domain.RecalcItemBurnRateEvent{
		AccountID: accountID,
		ItemID:    itemID,
	})
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal recalc item burn rate event.")
	}

	msg := contracts.AmqpMessage{
		Data: payload,
	}
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}

	outboxInput := messaging.OutboxMessageInput{
		ServiceName: "core-service",
		MessageType: string(contracts.CoreCmdRecalcItemBurnRate),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.CoreCmdRecalcItemBurnRate),
		Payload:     msg,
	}

	if _, err := outboxRepo.Create(ctx, outboxInput); err != nil {
		return apierror.NewInternalError(err, "Failed to create outbox message for recalc item burn rate.")
	}

	return nil
}
