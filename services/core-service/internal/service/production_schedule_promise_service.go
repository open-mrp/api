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
