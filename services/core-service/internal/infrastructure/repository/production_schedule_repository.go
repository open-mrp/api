package repository

import (
	"context"
	gosql "database/sql"
	"encoding/json"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/safeconv"
	"github.com/augno/api/shared/tracing"
)

var productionScheduleRepoTracer = tracing.GetTracer("core-service.production_schedule_repository")

type productionScheduleRepoImpl struct {
	queries *sqlc.Queries
}

func NewProductionScheduleRepo(queries *sqlc.Queries) domain.ProductionScheduleRepo {
	return &productionScheduleRepoImpl{queries: queries}
}

func scheduleCreatedAt(s *domain.ProductionSchedule) time.Time { return s.CreatedAt }
func scheduleID(s *domain.ProductionSchedule) string           { return s.ID }

// NextVersion reserves the next version number atomically.
//
// The old read-MAX-then-write pattern raced: two planners generating at once both read the same maximum and the second collided on production_schedule_account_version_key. The counter is seeded from existing rows on first use so an account that already has versions does not restart numbering at 1.
func (r *productionScheduleRepoImpl) NextVersion(ctx context.Context, accountID string) (int32, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.next_version")
	defer span.End()

	seedID, apiErr := id.GenID(id.SysPropertyIDPrefix, nil)
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	if err := r.queries.SeedProductionScheduleVersionCounter(ctx, sqlc.SeedProductionScheduleVersionCounterParams{
		ID:        seedID,
		AccountID: accountID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return 0, tracing.Trace(span, apiErr)
		}
	}

	allocID, apiErr := id.GenID(id.SysPropertyIDPrefix, nil)
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	result, err := r.queries.AllocateNextProductionScheduleVersion(ctx, sqlc.AllocateNextProductionScheduleVersionParams{
		ID:        allocID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	next, err := result.LastInsertId()
	if err != nil {
		return 0, tracing.Trace(span, apierror.NewInternalError(err, "Failed to allocate a schedule version."))
	}
	return safeconv.Int64ToInt32(next), nil
}

func (r *productionScheduleRepoImpl) Create(ctx context.Context, schedule *domain.ProductionSchedule) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.create")
	defer span.End()

	settings := schedule.SettingsSnapshot
	if len(settings) == 0 {
		settings = json.RawMessage("{}")
	}
	diagnostics := schedule.Diagnostics
	if len(diagnostics) == 0 {
		diagnostics = json.RawMessage("{}")
	}

	err := r.queries.CreateProductionSchedule(ctx, sqlc.CreateProductionScheduleParams{
		ID:                   schedule.ID,
		AccountID:            schedule.AccountID,
		Version:              schedule.Version,
		StatusCode:           schedule.StatusCode,
		Name:                 dtNullString(schedule.Name),
		PlanningAsOf:         schedule.PlanningAsOf,
		HorizonStartDate:     schedule.HorizonStartDate,
		HorizonEndDate:       schedule.HorizonEndDate,
		HorizonWeeks:         schedule.HorizonWeeks,
		FrozenWeeks:          schedule.FrozenWeeks,
		DemandBasisCode:      schedule.DemandBasisCode,
		GenerationSourceCode: schedule.GenerationSourceCode,
		SolverVersion:        schedule.SolverVersion,
		SettingsSnapshot:     []byte(settings),
		Diagnostics:          []byte(diagnostics),
		GeneratedByID:        dtNullString(schedule.GeneratedByID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *productionScheduleRepoImpl) CreateLines(ctx context.Context, accountID, scheduleID string, lines []*domain.ProductionScheduleLine) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.create_lines")
	defer span.End()

	for _, line := range lines {
		err := r.queries.CreateProductionScheduleLine(ctx, sqlc.CreateProductionScheduleLineParams{
			ID:                       line.ID,
			AccountID:                accountID,
			ProductionScheduleID:     scheduleID,
			WeekIndex:                line.WeekIndex,
			WeekStartDate:            line.WeekStartDate,
			MachineID:                line.MachineID,
			ProductionStepID:         dtNullString(line.ProductionStepID),
			DepartmentID:             dtNullString(line.DepartmentID),
			ItemID:                   line.ItemID,
			PlannedQuantity:          floatToDecimalString(line.PlannedQuantity),
			PlannedUnitID:            dtNullString(line.PlannedUnitID),
			PlannedLots:              line.PlannedLots,
			PlannedLotUnits:          floatToDecimalString(line.PlannedLotUnits),
			PlannedRunHours:          floatToDecimalString(line.PlannedRunHours),
			PlannedChangeoverMinutes: floatToDecimalString(line.PlannedChangeoverMinutes),
			SequenceIndex:            line.SequenceIndex,
			ProjectedOnHandBefore:    floatToDecimalString(line.ProjectedOnHandBefore),
			ProjectedOnHandAfter:     floatToDecimalString(line.ProjectedOnHandAfter),
			StatusCode:               line.StatusCode,
			SourceCode:               line.SourceCode,
			ReasonCode:               dtNullString(line.ReasonCode),
			IsFrozen:                 line.IsFrozen,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	return nil
}

func (r *productionScheduleRepoImpl) CreateItemPolicies(ctx context.Context, accountID, scheduleID string, policies []*domain.ProductionScheduleItemPolicy) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.create_item_policies")
	defer span.End()

	for _, p := range policies {
		err := r.queries.CreateProductionScheduleItemPolicy(ctx, sqlc.CreateProductionScheduleItemPolicyParams{
			ID:                      p.ID,
			AccountID:               accountID,
			ProductionScheduleID:    scheduleID,
			ItemID:                  p.ItemID,
			Sku:                     p.SKU,
			ProductionStepID:        dtNullString(p.ProductionStepID),
			PrimaryMachineID:        dtNullString(p.PrimaryMachineID),
			UnitID:                  dtNullString(p.UnitID),
			AnnualDemand:            floatToDecimalString(p.AnnualDemand),
			WeeklyDemand:            floatToDecimalString(p.WeeklyDemand),
			SecondsPerUnit:          floatToDecimalString(p.SecondsPerUnit),
			UnitCost:                floatToDecimalString(p.UnitCost),
			SetupCost:               floatToDecimalString(p.SetupCost),
			HoldingCost:             floatToDecimalString(p.HoldingCost),
			EoqUnits:                floatToDecimalString(p.EOQUnits),
			ConstraintLeadTimeWeeks: floatToDecimalString(p.ConstraintLeadTimeWeeks),
			FinishLeadTimeWeeks:     floatToDecimalString(p.FinishLeadTimeWeeks),
			SigmaWeeklyPooled:       floatToDecimalString(p.SigmaWeeklyPooled),
			SigmaDownstreamSum:      floatToDecimalString(p.SigmaDownstreamSum),
			SafetyStockPrimary:      floatToDecimalString(p.SafetyStockPrimary),
			SafetyStockDownstream:   floatToDecimalString(p.SafetyStockDownstream),
			ReorderPoint:            floatToDecimalString(p.ReorderPoint),
			OrderUpTo:               floatToDecimalString(p.OrderUpTo),
			OnHandEchelon:           floatToDecimalString(p.OnHandEchelon),
			OnHandGreige:            floatToDecimalString(p.OnHandGreige),
			AverageGreigeInventory:  floatToDecimalString(p.AverageGreigeInventory),
			MaxGreigeInventory:      floatToDecimalString(p.MaxGreigeInventory),
			WeeksOfCover:            floatToDecimalString(p.WeeksOfCover),
			ProjectedOnHand:         marshalProjection(p.ProjectedOnHand),
			AnnualRunHours:          floatToDecimalString(p.AnnualRunHours),
			AbcClass:                dtNullString(p.ABCClass),
			WasEoqCapped:            p.WasEOQCapped,
			WasCapacityStarved:      p.WasCapacityStarved,
			FulfillmentPolicyCode:   p.FulfillmentPolicyCode,
			PolicySourceCode:        p.PolicySourceCode,
			FirmDemandUnits:         floatToDecimalString(p.FirmDemandUnits),
			ForecastDemandUnits:     floatToDecimalString(p.ForecastDemandUnits),
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	return nil
}

func (r *productionScheduleRepoImpl) Get(ctx context.Context, params domain.GetProductionScheduleParams) (*domain.ProductionSchedule, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.get")
	defer span.End()

	row, err := r.queries.GetProductionSchedule(ctx, sqlc.GetProductionScheduleParams{
		AccountID: params.AccountID,
		ID:        params.ScheduleID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return mapScheduleRow(scheduleRowFields(row)), nil
}

func (r *productionScheduleRepoImpl) GetCurrent(ctx context.Context, accountID string, asOf time.Time) (*domain.ProductionSchedule, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.get_current")
	defer span.End()

	row, err := r.queries.GetCurrentProductionSchedule(ctx, sqlc.GetCurrentProductionScheduleParams{
		AccountID: accountID,
		AsOfDate:  asOf,
	})
	if err == gosql.ErrNoRows {
		// No published plan covering today is a normal state, not an error.
		return nil, nil
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return mapScheduleRow(scheduleRowFields(row)), nil
}

func (r *productionScheduleRepoImpl) List(ctx context.Context, params domain.ListProductionSchedulesParams) (*domain.ListProductionSchedulesResult, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.list")
	defer span.End()

	includeStatusFilter := len(params.StatusCodes) > 0
	searchQuery := dtSearchParam(params.Query)
	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListProductionSchedulesBackward(ctx, sqlc.ListProductionSchedulesBackwardParams{
				AccountID:           params.AccountID,
				IncludeStatusFilter: includeStatusFilter,
				StatusCodes:         params.StatusCodes,
				SearchQuery:         searchQuery,
				CursorCreatedAt:     cur.OccurredAt,
				CursorID:            cur.ID,
				Limit:               params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			schedules := make([]*domain.ProductionSchedule, len(rows))
			for i, row := range rows {
				schedules[i] = mapScheduleRow(scheduleRowFields(row))
			}
			result, pageInfo := pagination.BuildPageString(schedules, params.Limit, cursorDir, scheduleCreatedAt, scheduleID)
			return &domain.ListProductionSchedulesResult{Schedules: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListProductionSchedulesForward(ctx, sqlc.ListProductionSchedulesForwardParams{
			AccountID:           params.AccountID,
			IncludeStatusFilter: includeStatusFilter,
			StatusCodes:         params.StatusCodes,
			SearchQuery:         searchQuery,
			CursorCreatedAt:     gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:            gosql.NullString{String: cur.ID, Valid: true},
			Limit:               params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		schedules := make([]*domain.ProductionSchedule, len(rows))
		for i, row := range rows {
			schedules[i] = mapScheduleRow(scheduleRowFields(row))
		}
		result, pageInfo := pagination.BuildPageString(schedules, params.Limit, cursorDir, scheduleCreatedAt, scheduleID)
		return &domain.ListProductionSchedulesResult{Schedules: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListProductionSchedulesForward(ctx, sqlc.ListProductionSchedulesForwardParams{
		AccountID:           params.AccountID,
		IncludeStatusFilter: includeStatusFilter,
		StatusCodes:         params.StatusCodes,
		SearchQuery:         searchQuery,
		Limit:               params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	schedules := make([]*domain.ProductionSchedule, len(rows))
	for i, row := range rows {
		schedules[i] = mapScheduleRow(scheduleRowFields(row))
	}
	result, pageInfo := pagination.BuildPageString(schedules, params.Limit, cursorDir, scheduleCreatedAt, scheduleID)
	return &domain.ListProductionSchedulesResult{Schedules: result, PageInfo: pageInfo}, nil
}

func (r *productionScheduleRepoImpl) ListLines(ctx context.Context, params domain.ListProductionScheduleLinesParams) ([]*domain.ProductionScheduleLine, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.list_lines")
	defer span.End()

	weekIndex := gosql.NullInt32{}
	if params.WeekIndex != nil {
		weekIndex = gosql.NullInt32{Int32: *params.WeekIndex, Valid: true}
	}

	rows, err := r.queries.ListProductionScheduleLines(ctx, sqlc.ListProductionScheduleLinesParams{
		AccountID:            params.AccountID,
		ProductionScheduleID: params.ScheduleID,
		IncludeMachineFilter: len(params.MachineIDs) > 0,
		MachineIds:           params.MachineIDs,
		WeekIndex:            weekIndex,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	lines := make([]*domain.ProductionScheduleLine, len(rows))
	for i, row := range rows {
		line := &domain.ProductionScheduleLine{
			ID:                      row.ID,
			ProductionScheduleID:    row.ProductionScheduleID,
			WeekIndex:               row.WeekIndex,
			WeekStartDate:           row.WeekStartDate,
			MachineID:               row.MachineID,
			ItemID:                  row.ItemID,
			PlannedQuantity:         decimalToFloat64(row.PlannedQuantity),
			PlannedLots:             row.PlannedLots,
			PlannedLotUnits:         decimalToFloat64(row.PlannedLotUnits),
			PlannedUnitAbbreviation: nullStringPtr(row.PlannedUnitAbbreviation),
			// COALESCE over a LEFT JOIN comes back untyped, so these need the same coercion any aggregate column does.
			ReleasedBatchCount:       int64(decimalToFloat64(row.ReleasedBatchCount)),
			ScannedBatchCount:        int64(decimalToFloat64(row.ScannedBatchCount)),
			ScannedQuantity:          decimalToFloat64(row.ScannedQuantity),
			PlannedRunHours:          decimalToFloat64(row.PlannedRunHours),
			PlannedChangeoverMinutes: decimalToFloat64(row.PlannedChangeoverMinutes),
			SequenceIndex:            row.SequenceIndex,
			ProjectedOnHandBefore:    decimalToFloat64(row.ProjectedOnHandBefore),
			ProjectedOnHandAfter:     decimalToFloat64(row.ProjectedOnHandAfter),
			StatusCode:               row.StatusCode,
			SourceCode:               row.SourceCode,
			IsFrozen:                 row.IsFrozen,
			CreatedAt:                row.CreatedAt,
			UpdatedAt:                row.UpdatedAt,
		}
		if row.ProductionStepID.Valid {
			line.ProductionStepID = &row.ProductionStepID.String
		}
		if row.DepartmentID.Valid {
			line.DepartmentID = &row.DepartmentID.String
		}
		if row.PlannedUnitID.Valid {
			line.PlannedUnitID = &row.PlannedUnitID.String
		}
		if row.ReasonCode.Valid {
			line.ReasonCode = &row.ReasonCode.String
		}
		if row.ProductionRunID.Valid {
			line.ProductionRunID = &row.ProductionRunID.String
		}
		lines[i] = line
	}
	return lines, nil
}

func (r *productionScheduleRepoImpl) ListItemPolicies(ctx context.Context, accountID, scheduleID string) ([]*domain.ProductionScheduleItemPolicy, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.list_item_policies")
	defer span.End()

	rows, err := r.queries.ListProductionScheduleItemPolicies(ctx, sqlc.ListProductionScheduleItemPoliciesParams{
		AccountID:            accountID,
		ProductionScheduleID: scheduleID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	policies := make([]*domain.ProductionScheduleItemPolicy, len(rows))
	for i, row := range rows {
		p := &domain.ProductionScheduleItemPolicy{
			ID:                      row.ID,
			ProductionScheduleID:    row.ProductionScheduleID,
			ItemID:                  row.ItemID,
			SKU:                     row.Sku,
			AnnualDemand:            decimalToFloat64(row.AnnualDemand),
			WeeklyDemand:            decimalToFloat64(row.WeeklyDemand),
			SecondsPerUnit:          decimalToFloat64(row.SecondsPerUnit),
			UnitCost:                decimalToFloat64(row.UnitCost),
			SetupCost:               decimalToFloat64(row.SetupCost),
			HoldingCost:             decimalToFloat64(row.HoldingCost),
			EOQUnits:                decimalToFloat64(row.EoqUnits),
			ConstraintLeadTimeWeeks: decimalToFloat64(row.ConstraintLeadTimeWeeks),
			FinishLeadTimeWeeks:     decimalToFloat64(row.FinishLeadTimeWeeks),
			SigmaWeeklyPooled:       decimalToFloat64(row.SigmaWeeklyPooled),
			SigmaDownstreamSum:      decimalToFloat64(row.SigmaDownstreamSum),
			SafetyStockPrimary:      decimalToFloat64(row.SafetyStockPrimary),
			SafetyStockDownstream:   decimalToFloat64(row.SafetyStockDownstream),
			ReorderPoint:            decimalToFloat64(row.ReorderPoint),
			OrderUpTo:               decimalToFloat64(row.OrderUpTo),
			OnHandEchelon:           decimalToFloat64(row.OnHandEchelon),
			OnHandGreige:            decimalToFloat64(row.OnHandGreige),
			AverageGreigeInventory:  decimalToFloat64(row.AverageGreigeInventory),
			MaxGreigeInventory:      decimalToFloat64(row.MaxGreigeInventory),
			WeeksOfCover:            decimalToFloat64(row.WeeksOfCover),
			ProjectedOnHand:         unmarshalProjection(row.ProjectedOnHand),
			AnnualRunHours:          decimalToFloat64(row.AnnualRunHours),
			WasEOQCapped:            row.WasEoqCapped,
			WasCapacityStarved:      row.WasCapacityStarved,
			FulfillmentPolicyCode:   row.FulfillmentPolicyCode,
			PolicySourceCode:        row.PolicySourceCode,
			FirmDemandUnits:         decimalToFloat64(row.FirmDemandUnits),
			ForecastDemandUnits:     decimalToFloat64(row.ForecastDemandUnits),
			CreatedAt:               row.CreatedAt,
			UpdatedAt:               row.UpdatedAt,
		}
		if row.ProductionStepID.Valid {
			p.ProductionStepID = &row.ProductionStepID.String
		}
		p.UnitID = nullStringPtr(row.UnitID)
		p.UnitAbbreviation = nullStringPtr(row.UnitAbbreviation)
		if row.PrimaryMachineID.Valid {
			p.PrimaryMachineID = &row.PrimaryMachineID.String
		}
		if row.AbcClass.Valid {
			p.ABCClass = &row.AbcClass.String
		}
		policies[i] = p
	}
	return policies, nil
}

func (r *productionScheduleRepoImpl) Delete(ctx context.Context, accountID, scheduleID string) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.delete")
	defer span.End()

	// Children first: there are no enforced foreign keys, so nothing cascades and deleting the header alone would orphan every line and policy.
	if err := r.queries.DeleteProductionScheduleLines(ctx, sqlc.DeleteProductionScheduleLinesParams{
		AccountID:            accountID,
		ProductionScheduleID: scheduleID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	if err := r.queries.DeleteProductionScheduleItemPolicies(ctx, sqlc.DeleteProductionScheduleItemPoliciesParams{
		AccountID:            accountID,
		ProductionScheduleID: scheduleID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	err := r.queries.DeleteProductionSchedule(ctx, sqlc.DeleteProductionScheduleParams{
		AccountID: accountID,
		ID:        scheduleID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// scheduleRowFields is the shared column set every schedule query selects; sqlc emits a distinct row type per query with identical shape, so each funnels through here.
type scheduleRowFields struct {
	ID                    string
	AccountID             string
	Version               int32
	StatusCode            string
	Name                  gosql.NullString
	PlanningAsOf          time.Time
	HorizonStartDate      time.Time
	HorizonEndDate        time.Time
	HorizonWeeks          int32
	FrozenWeeks           int32
	FrozenThroughDate     gosql.NullTime
	DemandBasisCode       string
	GenerationSourceCode  string
	SolverVersion         string
	SettingsSnapshot      json.RawMessage
	Diagnostics           json.RawMessage
	ErrorMessage          gosql.NullString
	FrozenLineCount       int32
	FrozenPlannedQuantity string
	GeneratedByID         gosql.NullString
	PublishedByID         gosql.NullString
	PublishedAt           gosql.NullTime
	SupersededByID        gosql.NullString
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func mapScheduleRow(f scheduleRowFields) *domain.ProductionSchedule {
	s := &domain.ProductionSchedule{
		ID:                    f.ID,
		AccountID:             f.AccountID,
		Version:               f.Version,
		StatusCode:            f.StatusCode,
		PlanningAsOf:          f.PlanningAsOf,
		HorizonStartDate:      f.HorizonStartDate,
		HorizonEndDate:        f.HorizonEndDate,
		HorizonWeeks:          f.HorizonWeeks,
		FrozenWeeks:           f.FrozenWeeks,
		DemandBasisCode:       f.DemandBasisCode,
		GenerationSourceCode:  f.GenerationSourceCode,
		SolverVersion:         f.SolverVersion,
		SettingsSnapshot:      f.SettingsSnapshot,
		Diagnostics:           f.Diagnostics,
		FrozenLineCount:       f.FrozenLineCount,
		FrozenPlannedQuantity: decimalToFloat64(f.FrozenPlannedQuantity),
		CreatedAt:             f.CreatedAt,
		UpdatedAt:             f.UpdatedAt,
	}
	if f.Name.Valid {
		s.Name = &f.Name.String
	}
	if f.FrozenThroughDate.Valid {
		s.FrozenThroughDate = &f.FrozenThroughDate.Time
	}
	if f.ErrorMessage.Valid {
		s.ErrorMessage = &f.ErrorMessage.String
	}
	if f.GeneratedByID.Valid {
		s.GeneratedByID = &f.GeneratedByID.String
	}
	if f.PublishedByID.Valid {
		s.PublishedByID = &f.PublishedByID.String
	}
	if f.PublishedAt.Valid {
		s.PublishedAt = &f.PublishedAt.Time
	}
	if f.SupersededByID.Valid {
		s.SupersededByID = &f.SupersededByID.String
	}
	return s
}

func (r *productionScheduleRepoImpl) DeleteItemPolicies(ctx context.Context, accountID, scheduleID string) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.delete_item_policies")
	defer span.End()

	err := r.queries.DeleteProductionScheduleItemPolicies(ctx, sqlc.DeleteProductionScheduleItemPoliciesParams{
		AccountID:            accountID,
		ProductionScheduleID: scheduleID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// ReplaceFinishedPolicies rewrites a version's finished-goods targets.
//
// Wholesale like the item policies: the snapshot describes one solve, and a partial update could leave a finished SKU's target describing demand the version never saw.
func (r *productionScheduleRepoImpl) ReplaceFinishedPolicies(ctx context.Context, accountID, scheduleID string, policies []*domain.ProductionScheduleFinishedPolicy) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.replace_finished_policies")
	defer span.End()

	if err := r.queries.DeleteProductionScheduleFinishedPolicies(ctx, sqlc.DeleteProductionScheduleFinishedPoliciesParams{
		AccountID:            accountID,
		ProductionScheduleID: scheduleID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	for _, p := range policies {
		if err := r.queries.CreateProductionScheduleFinishedPolicy(ctx, sqlc.CreateProductionScheduleFinishedPolicyParams{
			ID:                   p.ID,
			AccountID:            accountID,
			ProductionScheduleID: scheduleID,
			ItemID:               p.ItemID,
			Sku:                  p.SKU,
			GreigeItemID:         p.GreigeItemID,
			GreigeSku:            p.GreigeSKU,
			ProductLineID:        dtNullString(p.ProductLineID),
			AnnualDemand:         floatToDecimalString(p.AnnualDemand),
			WeeklyDemand:         floatToDecimalString(p.WeeklyDemand),
			SigmaWeekly:          floatToDecimalString(p.SigmaWeekly),
			SafetyStock:          floatToDecimalString(p.SafetyStock),
			ReorderPoint:         floatToDecimalString(p.ReorderPoint),
			OnHand:               floatToDecimalString(p.OnHand),
			WeeksOfCover:         floatToDecimalString(p.WeeksOfCover),
		}); err != nil {
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return tracing.Trace(span, apiErr)
			}
		}
	}

	return nil
}

func (r *productionScheduleRepoImpl) ListFinishedPolicies(ctx context.Context, accountID, scheduleID string) ([]*domain.ProductionScheduleFinishedPolicy, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.list_finished_policies")
	defer span.End()

	rows, err := r.queries.ListProductionScheduleFinishedPolicies(ctx, sqlc.ListProductionScheduleFinishedPoliciesParams{
		AccountID:            accountID,
		ProductionScheduleID: scheduleID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]*domain.ProductionScheduleFinishedPolicy, len(rows))
	for i, row := range rows {
		policy := &domain.ProductionScheduleFinishedPolicy{
			ID:                   row.ID,
			ProductionScheduleID: row.ProductionScheduleID,
			ItemID:               row.ItemID,
			SKU:                  row.Sku,
			GreigeItemID:         row.GreigeItemID,
			GreigeSKU:            row.GreigeSku,
			AnnualDemand:         decimalToFloat64(row.AnnualDemand),
			WeeklyDemand:         decimalToFloat64(row.WeeklyDemand),
			SigmaWeekly:          decimalToFloat64(row.SigmaWeekly),
			SafetyStock:          decimalToFloat64(row.SafetyStock),
			ReorderPoint:         decimalToFloat64(row.ReorderPoint),
			OnHand:               decimalToFloat64(row.OnHand),
			WeeksOfCover:         decimalToFloat64(row.WeeksOfCover),
			CreatedAt:            row.CreatedAt,
			UpdatedAt:            row.UpdatedAt,
		}
		if row.ProductLineID.Valid {
			policy.ProductLineID = &row.ProductLineID.String
		}
		out[i] = policy
	}
	return out, nil
}

// marshalProjection stores the week-by-week position as JSON. A nil curve stores NULL rather than "null", so a version generated before projections were kept reads as absent instead of as an empty plan.
func marshalProjection(values []float64) json.RawMessage {
	if len(values) == 0 {
		return nil
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return nil
	}
	return encoded
}

// unmarshalProjection reads it back, treating anything unparseable as absent: a broken curve should leave the weeks unexplained, not report a stockout that is not real.
func unmarshalProjection(raw any) []float64 {
	var text string
	switch value := raw.(type) {
	case nil:
		return nil
	case string:
		text = value
	case []byte:
		text = string(value)
	default:
		return nil
	}
	if text == "" {
		return nil
	}
	var out []float64
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil
	}
	return out
}

// ReplaceLineOrders rewrites a version's campaign-to-order links.
//
// Wholesale rather than incremental: the links are a pure function of the plan and the order book, so a partial update could leave a campaign earmarked for an order that has since shipped.
func (r *productionScheduleRepoImpl) ReplaceLineOrders(ctx context.Context, accountID, scheduleID string, links []domain.CreateLineOrderParams) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.replace_line_orders")
	defer span.End()

	if err := r.queries.DeleteProductionScheduleLineOrders(ctx, sqlc.DeleteProductionScheduleLineOrdersParams{
		AccountID:            accountID,
		ProductionScheduleID: scheduleID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	for _, link := range links {
		linkID, apiErr := id.GenID(id.ProductionScheduleLineOrderIDPrefix, nil)
		if apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
		err := r.queries.CreateProductionScheduleLineOrder(ctx, sqlc.CreateProductionScheduleLineOrderParams{
			ID:                       linkID,
			AccountID:                accountID,
			ProductionScheduleID:     scheduleID,
			ProductionScheduleLineID: link.ProductionScheduleLineID,
			SalesOrderID:             link.SalesOrderID,
			SalesOrderLineID:         link.SalesOrderLineID,
			AllocatedQuantity:        floatToDecimalString(link.AllocatedQuantity),
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	return nil
}

func (r *productionScheduleRepoImpl) ListLineOrders(ctx context.Context, accountID, scheduleID string) ([]*domain.ProductionScheduleLineOrder, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.list_line_orders")
	defer span.End()

	rows, err := r.queries.ListProductionScheduleLineOrders(ctx, sqlc.ListProductionScheduleLineOrdersParams{
		AccountID:            accountID,
		ProductionScheduleID: scheduleID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]*domain.ProductionScheduleLineOrder, 0, len(rows))
	for _, row := range rows {
		link := &domain.ProductionScheduleLineOrder{
			ID:                       row.ID,
			ProductionScheduleLineID: row.ProductionScheduleLineID,
			SalesOrderID:             row.SalesOrderID,
			SalesOrderNumber:         row.SalesOrderNumber,
			SalesOrderLineID:         row.SalesOrderLineID,
			AllocatedQuantity:        decimalToFloat64(row.AllocatedQuantity),
			ItemID:                   row.ItemID,
			SKU:                      row.Sku,
			WeekIndex:                row.WeekIndex,
			MachineID:                row.MachineID,
		}
		if row.ShipByDate.Valid {
			link.ShipByDate = &row.ShipByDate.Time
		}
		out = append(out, link)
	}
	return out, nil
}
