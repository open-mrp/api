package service

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/scheduling"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

// AnalyzeDeliveryPerformance measures what was promised against what was shipped.
//
// The commitment measured against is the one stamped on the order when it was issued, not one recomputed now. An order whose customer has since renegotiated a shorter lead time is still judged against what it was actually promised — which is the whole reason the commitment is materialized rather than derived.
func (s *analyticsSvcImpl) AnalyzeDeliveryPerformance(ctx context.Context, params domain.AnalyzeDeliveryPerformanceParams) (*domain.DeliveryPerformanceResult, *apierror.APIError) {
	ctx, span := analyticsSvcTracer.Start(ctx, "service.analytics.analyze_delivery_performance")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsInternalActor(); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// Delivery performance is a property of the order book, so it is gated on sales orders rather than on the schedule.
	if apiErr := identity.CheckHasPermission(types.PermissionDomainSalesOrders, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !params.EndDate.After(params.StartDate) {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("The end date must be after the start date.", "ends_at"))
	}
	accountID := identity.Target.AccountID

	repo := s.repos.NewProductionScheduleInputRepo()
	outcomes, apiErr := repo.ListDeliveryOutcomes(ctx, accountID, params.StartDate, params.EndDate, params.DeliveryFilters)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	uncommitted, apiErr := repo.CountUncommittedOrders(ctx, accountID, params.StartDate, params.EndDate, params.DeliveryFilters)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// One read of the clock for every derived figure, so a breakdown and the total it belongs to can never disagree about which orders are late right now.
	asOf := time.Now().UTC()
	periods, overall := scheduling.AnalyzeDeliveryPerformance(outcomes, deliveryBucketer(params.Granularity), asOf)

	return &domain.DeliveryPerformanceResult{
		Overall:  overall,
		Periods:  periods,
		Backlog:  scheduling.AnalyzeBacklogAging(outcomes, asOf),
		Lateness: scheduling.AnalyzeLatenessDistribution(outcomes, asOf),
		// Every breakdown is the same set of outcomes sliced a different way rather than re-read per dimension: four queries would let four numbers disagree, and the whole point of a drilldown is that it adds up to the headline.
		ByCustomer:            scheduling.AnalyzeDeliveryBreakdown(outcomes, asOf, scheduling.ByCustomer),
		ByCustomerGroup:       scheduling.AnalyzeDeliveryBreakdown(outcomes, asOf, scheduling.ByCustomerGroup),
		ByProductLine:         scheduling.AnalyzeDeliveryBreakdown(outcomes, asOf, scheduling.ByProductLine),
		ByCommitmentSource:    scheduling.AnalyzeDeliveryBreakdown(outcomes, asOf, scheduling.ByCommitmentSource),
		UncommittedOrderCount: uncommitted,
	}, nil
}

// deliveryBucketer maps a commitment date onto the period it belongs to.
//
// Bucketed by the date promised rather than the date shipped: the question is how well a period's commitments were met, and an order promised in March and shipped in May is March's miss rather than May's.
func deliveryBucketer(granularity string) func(time.Time) time.Time {
	switch constants.DeliveryGranularity(granularity) {
	case constants.DeliveryGranularityDay:
		return func(t time.Time) time.Time {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		}
	case constants.DeliveryGranularityMonth:
		return func(t time.Time) time.Time {
			return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
		}
	default:
		// Weeks start Monday, matching how the schedule itself is bucketed, so a delivery week and a plan week name the same seven days.
		return func(t time.Time) time.Time {
			day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			offset := (int(day.Weekday()) + 6) % 7
			return day.AddDate(0, 0, -offset)
		}
	}
}
