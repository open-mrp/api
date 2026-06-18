package mediator

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
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

// RecalculateFromHistory updates the item's burn_rate from consumption change logs over the last 30 days. No-op when there is insufficient history.
//
//  1. Load the item and resolve its category's base unit.
//  2. List the item's consumption change logs; no-op when fewer than two exist.
//  3. Sum the absolute consumption quantities, converting each to the base unit.
//  4. Divide the total by the days elapsed between the first and last log.
//  5. Persist the resulting per-day rate to the item's burn rate.
func (m *burnRateMedImpl) RecalculateFromHistory(ctx context.Context, accountID, itemID string) *apierror.APIError {
	ctx, span := burnRateMedTracer.Start(ctx, "mediator.burn_rate.recalculate_from_history")
	defer span.End()

	itemRepo := m.repos.NewItemRepo()
	item, apiErr := itemRepo.Get(ctx, domain.GetItemParams{
		AccountID: accountID,
		ItemID:    itemID,
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	baseUnitID, apiErr := itemRepo.GetCategoryBaseUnitID(ctx, item.ItemCategoryID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	logs, apiErr := itemRepo.ListConsumptionChangeLogsForBurnRate(ctx, accountID, itemID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if len(logs) < 2 {
		return nil
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
		return nil
	}

	timeSpanDays := burnRateTimeSpanDays(logs[0].CreatedAt, logs[len(logs)-1].CreatedAt)
	if timeSpanDays <= 0 {
		return nil
	}

	burnRateMeasure := totalConsumption.Div(decimal.NewFromFloat(timeSpanDays))
	valueStr := burnRateMeasure.String()

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

// MaybeRecalculateAfterConsumption recalculates burn rate when a consumption change log was recorded. Errors are traced but do not fail the caller's primary operation.
func MaybeRecalculateAfterConsumption(
	ctx context.Context,
	meds domain.Mediators,
	accountID, itemID string,
	delta decimal.Decimal,
	actionType string,
) {
	if delta.GreaterThanOrEqual(decimal.Zero) {
		return
	}
	if actionType != "scan" && actionType != "user_correction" {
		return
	}
	ctx, span := burnRateMedTracer.Start(ctx, "mediator.burn_rate.maybe_recalculate_after_consumption")
	defer span.End()
	if apiErr := meds.BurnRate.RecalculateFromHistory(ctx, accountID, itemID); apiErr != nil {
		tracing.Trace(span, apiErr)
	}
}

// IncludesItemBurnRate reports whether API includes request item.burn_rate.
func IncludesItemBurnRate(includes []string) bool {
	for _, inc := range includes {
		if inc == "item.burn_rate" {
			return true
		}
	}
	return false
}

// RefreshItemBurnRateAfterGet recalculates burn rate when item.burn_rate was included, then reloads item.BurnRate.
func RefreshItemBurnRateAfterGet(
	ctx context.Context,
	repos domain.RepoFactory,
	meds domain.Mediators,
	accountID string,
	item *domain.Item,
	includes []string,
) {
	if !IncludesItemBurnRate(includes) || item == nil {
		return
	}
	ctx, span := burnRateMedTracer.Start(ctx, "mediator.burn_rate.refresh_after_get")
	defer span.End()
	if apiErr := meds.BurnRate.RecalculateFromHistory(ctx, accountID, item.ID); apiErr != nil {
		tracing.Trace(span, apiErr)
		return
	}
	rate, apiErr := repos.NewRateRepo().Get(ctx, item.BurnRateID)
	if apiErr != nil {
		tracing.Trace(span, apiErr)
		return
	}
	item.BurnRate = rate
}
