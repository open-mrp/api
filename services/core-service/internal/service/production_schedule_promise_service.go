package service

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/scheduling"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

// ListAtRiskOrders returns the commitments a persisted version does not meet, with the campaigns earmarked for each.
//
// Read from the version's own diagnostics and its stored campaign-to-order links rather than re-solved. A version is a record of what was decided; re-solving to answer "what is at risk" would answer about a plan nobody published.
func (s *productionScheduleSvcImpl) ListAtRiskOrders(ctx context.Context, scheduleID string) ([]*domain.ScheduleOrderCoverage, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.list_at_risk_orders")
	defer span.End()

	identity, apiErr := s.readIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID

	repo := s.repos.NewProductionScheduleRepo()
	schedule, apiErr := repo.Get(ctx, domain.GetProductionScheduleParams{AccountID: accountID, ScheduleID: scheduleID})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	atRisk, apiErr := decodeAtRiskDiagnostics(schedule.Diagnostics)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(atRisk) == 0 {
		return nil, nil
	}

	links, apiErr := repo.ListLineOrders(ctx, accountID, scheduleID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// An at-risk order can still be partly built, so the campaigns covering the rest are shown beside the shortfall — "we build 300 of the 500" is a different conversation from "we build none of it".
	coveringByOrder := map[string][]domain.ScheduleOrderCoverageLine{}
	shipByOrder := map[string]*time.Time{}
	for _, link := range links {
		coveringByOrder[link.SalesOrderID] = append(coveringByOrder[link.SalesOrderID], domain.ScheduleOrderCoverageLine{
			ProductionScheduleLineID: link.ProductionScheduleLineID,
			WeekIndex:                link.WeekIndex,
			MachineID:                link.MachineID,
			AllocatedQuantity:        link.AllocatedQuantity,
		})
		if link.ShipByDate != nil {
			shipByOrder[link.SalesOrderID] = link.ShipByDate
		}
	}

	out := make([]*domain.ScheduleOrderCoverage, 0, len(atRisk))
	for _, row := range atRisk {
		coverage := &domain.ScheduleOrderCoverage{
			SalesOrderID:     row.SalesOrderID,
			SalesOrderNumber: row.SalesOrderNumber,
			ItemID:           row.ItemID,
			SKU:              row.SKU,
			UnitsAtRisk:      row.Units,
			DueWeek:          row.DueWeek,
			ReasonCode:       row.Reason,
			CoveringLines:    coveringByOrder[row.SalesOrderID],
			ShipByDate:       shipByOrder[row.SalesOrderID],
		}
		out = append(out, coverage)
	}

	// Soonest first, which is the order they have to be dealt with.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].DueWeek != out[j].DueWeek {
			return out[i].DueWeek < out[j].DueWeek
		}
		return out[i].SalesOrderNumber < out[j].SalesOrderNumber
	})

	return out, nil
}

// QuotePromiseDate says the earliest date the published plan could ship a quantity of an item.
//
// Capable-to-promise, drawn from the published version rather than a fresh solve: a date offered to a customer has to come from the plan the floor is working to, not from one that exists only inside this request. Supply already earmarked for other orders is consumed first — a date backed by stock somebody else is owed is not a date.
func (s *productionScheduleSvcImpl) QuotePromiseDate(ctx context.Context, itemID string, quantity float64) (*domain.PromiseDateQuote, *apierror.APIError) {
	ctx, span := productionScheduleSvcTracer.Start(ctx, "service.production_schedule.quote_promise_date")
	defer span.End()

	identity, apiErr := s.readIdentity(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainProductionSchedules, types.ActionRead); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if quantity <= 0 {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam("Quantity must be greater than zero.", "quantity"))
	}
	accountID := identity.Target.AccountID

	repo := s.repos.NewProductionScheduleRepo()
	published, apiErr := repo.GetCurrent(ctx, accountID, time.Now().UTC())
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if published == nil {
		// Nothing is published, so there is no plan to promise from. Saying so beats inventing a date from a draft the floor is not working to.
		return nil, tracing.Trace(span, apierror.NewValidationError("No published schedule covers today, so no delivery date can be quoted."))
	}

	lines, apiErr := repo.ListLines(ctx, domain.ListProductionScheduleLinesParams{AccountID: accountID, ScheduleID: published.ID})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	campaigns := make([]scheduling.Campaign, 0, len(lines))
	for _, l := range lines {
		if l.ItemID != itemID {
			continue
		}
		campaigns = append(campaigns, scheduling.Campaign{
			ItemID:    l.ItemID,
			MachineID: l.MachineID,
			WeekIndex: int(l.WeekIndex),
			Units:     l.PlannedQuantity,
		})
	}

	// What is already owed out of this plan, so the quote is of what remains rather than of the whole plan.
	links, apiErr := repo.ListLineOrders(ctx, accountID, published.ID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	committed := scheduling.FirmSchedule{ByItemWeek: map[string][]float64{}}
	horizonWeeks := int(published.HorizonWeeks)
	for _, link := range links {
		if link.ItemID != itemID {
			continue
		}
		if committed.ByItemWeek[itemID] == nil {
			committed.ByItemWeek[itemID] = make([]float64, horizonWeeks)
		}
		if int(link.WeekIndex) < horizonWeeks {
			committed.ByItemWeek[itemID][link.WeekIndex] += link.AllocatedQuantity
		}
	}

	onHand, apiErr := s.repos.NewProductionScheduleInputRepo().GetEchelonOnHand(ctx, accountID, []string{itemID})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	quote := &domain.PromiseDateQuote{
		ItemID:                    itemID,
		Quantity:                  quantity,
		ProductionScheduleID:      published.ID,
		ProductionScheduleVersion: published.Version,
	}

	week, ok := scheduling.EarliestPromiseWeek(itemID, quantity, campaigns, committed, onHand, horizonWeeks)
	if !ok {
		// The horizon simply does not reach. Reported rather than extrapolated: a plan that runs thirteen weeks cannot speak for the fourteenth.
		return quote, nil
	}

	// The constraint finishes in that week; finishing still has to run before it can ship.
	settings, apiErr := s.loadEffectiveSettings(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	shipWeek := week + int(scheduling.CeilWeeks(settings.Settings.FinishLeadTimeWeeks))
	shipDate := published.HorizonStartDate.AddDate(0, 0, shipWeek*7)

	quote.IsPromisable = true
	quote.EarliestWeekIndex = &week
	quote.EarliestShipDate = &shipDate
	return quote, nil
}

// decodeAtRiskDiagnostics reads the at-risk orders out of a version's stored diagnostics blob.
//
// The diagnostics are stored as JSON so a plan stays explainable after the code that produced it moves. A version solved before at-risk orders existed simply has no such key, which decodes to nothing rather than an error — an old plan has no commitments to report, not a broken one.
func decodeAtRiskDiagnostics(raw json.RawMessage) ([]scheduling.AtRiskOrder, *apierror.APIError) {
	if len(raw) == 0 {
		return nil, nil
	}
	var envelope struct {
		AtRiskOrders []scheduling.AtRiskOrder `json:"at_risk_orders"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, apierror.NewInternalError(err, "Could not read the schedule's diagnostics.")
	}
	return envelope.AtRiskOrders, nil
}
