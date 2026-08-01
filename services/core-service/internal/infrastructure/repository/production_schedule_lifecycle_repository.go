package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/safeconv"
	"github.com/augno/api/shared/tracing"
)

func deviationCreatedAt(d *domain.ProductionScheduleDeviation) time.Time { return d.CreatedAt }
func deviationID(d *domain.ProductionScheduleDeviation) string           { return d.ID }

func psNullInt32(v *int32) gosql.NullInt32 {
	if v == nil {
		return gosql.NullInt32{}
	}
	return gosql.NullInt32{Int32: *v, Valid: true}
}

func psNullDecimal(v *float64) gosql.NullString {
	if v == nil {
		return gosql.NullString{}
	}
	return gosql.NullString{String: floatToDecimalString(*v), Valid: true}
}

func (r *productionScheduleRepoImpl) ListScheduleDeviationTypes(ctx context.Context) ([]*domain.ScheduleDeviationType, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.list_deviation_types")
	defer span.End()

	rows, err := r.queries.ListScheduleDeviationTypes(ctx)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	types := make([]*domain.ScheduleDeviationType, len(rows))
	for i, row := range rows {
		types[i] = &domain.ScheduleDeviationType{
			ID:        row.ID,
			Code:      row.Code,
			Name:      row.Name,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
	}
	return types, nil
}

func (r *productionScheduleRepoImpl) GetLine(ctx context.Context, accountID, lineID string) (*domain.ProductionScheduleLine, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.get_line")
	defer span.End()

	row, err := r.queries.GetProductionScheduleLine(ctx, sqlc.GetProductionScheduleLineParams{
		AccountID: accountID,
		ID:        lineID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	line := &domain.ProductionScheduleLine{
		ID:                       row.ID,
		ProductionScheduleID:     row.ProductionScheduleID,
		WeekIndex:                row.WeekIndex,
		WeekStartDate:            row.WeekStartDate,
		MachineID:                row.MachineID,
		ItemID:                   row.ItemID,
		PlannedQuantity:          decimalToFloat64(row.PlannedQuantity),
		PlannedLots:              row.PlannedLots,
		PlannedLotUnits:          decimalToFloat64(row.PlannedLotUnits),
		PlannedUnitAbbreviation:  nullStringPtr(row.PlannedUnitAbbreviation),
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
	return line, nil
}

func (r *productionScheduleRepoImpl) UpdateLine(ctx context.Context, params domain.UpdateLineRepoParams) (*domain.ProductionScheduleLine, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.update_line")
	defer span.End()

	weekStart := gosql.NullTime{}
	if params.WeekStartDate != nil {
		weekStart = gosql.NullTime{Time: *params.WeekStartDate, Valid: true}
	}

	err := r.queries.UpdateProductionScheduleLine(ctx, sqlc.UpdateProductionScheduleLineParams{
		AccountID:       params.AccountID,
		ID:              params.LineID,
		MachineID:       dtNullString(params.MachineID),
		WeekIndex:       psNullInt32(params.WeekIndex),
		WeekStartDate:   weekStart,
		PlannedQuantity: psNullDecimal(params.PlannedQuantity),
		PlannedLots:     psNullInt32(params.PlannedLots),
		PlannedRunHours: psNullDecimal(params.PlannedRunHours),
		SequenceIndex:   psNullInt32(params.SequenceIndex),
		StatusCode:      dtNullString(params.StatusCode),
		ReasonCode:      dtNullString(params.ReasonCode),
		ClearReasonCode: params.ClearReasonCode,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.GetLine(ctx, params.AccountID, params.LineID)
}

func (r *productionScheduleRepoImpl) DeleteLine(ctx context.Context, accountID, lineID string) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.delete_line")
	defer span.End()

	err := r.queries.DeleteProductionScheduleLine(ctx, sqlc.DeleteProductionScheduleLineParams{
		AccountID: accountID,
		ID:        lineID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *productionScheduleRepoImpl) NextSequenceIndex(ctx context.Context, accountID, scheduleID string, weekIndex int32) (int32, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.next_sequence_index")
	defer span.End()

	maxIndex, err := r.queries.GetMaxSequenceIndexForWeek(ctx, sqlc.GetMaxSequenceIndexForWeekParams{
		AccountID:            accountID,
		ProductionScheduleID: scheduleID,
		WeekIndex:            weekIndex,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	// The query returns -1 for an empty week, so a hand-added line lands at 0. MAX() surfaces as interface{}, and the driver may hand back []byte, so this goes through the decimal-aware converter instead of a bare type assertion.
	return int32(decimalToFloat64(maxIndex)) + 1, nil
}

func (r *productionScheduleRepoImpl) CreateDeviation(ctx context.Context, id string, deviation *domain.ProductionScheduleDeviation) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.create_deviation")
	defer span.End()

	err := r.queries.CreateProductionScheduleDeviation(ctx, sqlc.CreateProductionScheduleDeviationParams{
		ID:                       id,
		AccountID:                deviation.AccountID,
		ProductionScheduleID:     deviation.ProductionScheduleID,
		ProductionScheduleLineID: dtNullString(deviation.ProductionScheduleLineID),
		DeviationTypeCode:        deviation.DeviationTypeCode,
		IsFrozenWeek:             deviation.IsFrozenWeek,
		WeekIndex:                psNullInt32(deviation.WeekIndex),
		MachineID:                dtNullString(deviation.MachineID),
		ItemID:                   dtNullString(deviation.ItemID),
		BeforeJson:               deviation.BeforeJSON,
		AfterJson:                deviation.AfterJSON,
		DeltaQuantity:            floatToDecimalString(deviation.DeltaQuantity),
		DeltaRunHours:            floatToDecimalString(deviation.DeltaRunHours),
		ReasonCode:               dtNullString(deviation.ReasonCode),
		ReasonNote:               dtNullString(deviation.ReasonNote),
		ActorID:                  deviation.ActorID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// deviationFields is the shared column set both keyset directions select. sqlc emits a distinct row struct per query with an identical shape.
type deviationFields struct {
	ID                       string
	AccountID                string
	ProductionScheduleID     string
	ProductionScheduleLineID gosql.NullString
	DeviationTypeCode        string
	IsFrozenWeek             bool
	WeekIndex                gosql.NullInt32
	MachineID                gosql.NullString
	ItemID                   gosql.NullString
	// The queries CAST these to text, which sqlc surfaces as interface{}; the driver hands back []byte for a value and nil for a missing snapshot.
	BeforeJson    interface{}
	AfterJson     interface{}
	DeltaQuantity string
	DeltaRunHours string
	ReasonCode    gosql.NullString
	ReasonNote    gosql.NullString
	ActorID       string
	CreatedAt     time.Time
}

// aggregatedStringOf normalizes a varchar column that arrived through an aggregate. sqlc types MIN()/MAX() over a string column as interface{}, and the driver hands back []byte rather than string, so a direct assertion would silently yield "".
func aggregatedStringOf(v interface{}) string {
	switch typed := v.(type) {
	case nil:
		return ""
	case []byte:
		return string(typed)
	case string:
		return typed
	default:
		return ""
	}
}

// jsonBytesOf normalizes a CAST-to-text JSON column. Nil means the snapshot is genuinely absent — a removed line has no after, an added one has no before.
func jsonBytesOf(v interface{}) []byte {
	switch typed := v.(type) {
	case nil:
		return nil
	case []byte:
		return typed
	case string:
		return []byte(typed)
	default:
		return nil
	}
}

func mapDeviationRow(f deviationFields) *domain.ProductionScheduleDeviation {
	d := &domain.ProductionScheduleDeviation{
		ID:                   f.ID,
		AccountID:            f.AccountID,
		ProductionScheduleID: f.ProductionScheduleID,
		DeviationTypeCode:    f.DeviationTypeCode,
		IsFrozenWeek:         f.IsFrozenWeek,

		DeltaQuantity: decimalToFloat64(f.DeltaQuantity),
		DeltaRunHours: decimalToFloat64(f.DeltaRunHours),
		ActorID:       f.ActorID,
		CreatedAt:     f.CreatedAt,
	}
	if f.ProductionScheduleLineID.Valid {
		d.ProductionScheduleLineID = &f.ProductionScheduleLineID.String
	}
	if f.WeekIndex.Valid {
		d.WeekIndex = &f.WeekIndex.Int32
	}
	if f.MachineID.Valid {
		d.MachineID = &f.MachineID.String
	}
	if f.ItemID.Valid {
		d.ItemID = &f.ItemID.String
	}
	if f.ReasonCode.Valid {
		d.ReasonCode = &f.ReasonCode.String
	}
	if f.ReasonNote.Valid {
		d.ReasonNote = &f.ReasonNote.String
	}
	d.BeforeJSON = jsonBytesOf(f.BeforeJson)
	d.AfterJSON = jsonBytesOf(f.AfterJson)
	return d
}

func (r *productionScheduleRepoImpl) ListDeviations(ctx context.Context, params domain.ListProductionScheduleDeviationsParams) (*domain.ListProductionScheduleDeviationsResult, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.list_deviations")
	defer span.End()

	frozenOnly := doNullBool(params.FrozenOnly)
	searchQuery := dtSearchParam(params.Query)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListProductionScheduleDeviationsBackward(ctx, sqlc.ListProductionScheduleDeviationsBackwardParams{
				AccountID:            params.AccountID,
				ProductionScheduleID: params.ScheduleID,
				FrozenOnly:           frozenOnly,
				SearchQuery:          searchQuery,
				CursorCreatedAt:      cur.OccurredAt,
				CursorID:             cur.ID,
				Limit:                params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			deviations := make([]*domain.ProductionScheduleDeviation, len(rows))
			for i, row := range rows {
				deviations[i] = mapDeviationRow(deviationFields(row))
			}
			result, pageInfo := pagination.BuildPageString(deviations, params.Limit, cursorDir, deviationCreatedAt, deviationID)
			return &domain.ListProductionScheduleDeviationsResult{Deviations: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListProductionScheduleDeviationsForward(ctx, sqlc.ListProductionScheduleDeviationsForwardParams{
			AccountID:            params.AccountID,
			ProductionScheduleID: params.ScheduleID,
			FrozenOnly:           frozenOnly,
			SearchQuery:          searchQuery,
			CursorCreatedAt:      gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:             gosql.NullString{String: cur.ID, Valid: true},
			Limit:                params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		deviations := make([]*domain.ProductionScheduleDeviation, len(rows))
		for i, row := range rows {
			deviations[i] = mapDeviationRow(deviationFields(row))
		}
		result, pageInfo := pagination.BuildPageString(deviations, params.Limit, cursorDir, deviationCreatedAt, deviationID)
		return &domain.ListProductionScheduleDeviationsResult{Deviations: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListProductionScheduleDeviationsForward(ctx, sqlc.ListProductionScheduleDeviationsForwardParams{
		AccountID:            params.AccountID,
		ProductionScheduleID: params.ScheduleID,
		FrozenOnly:           frozenOnly,
		SearchQuery:          searchQuery,
		Limit:                params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	deviations := make([]*domain.ProductionScheduleDeviation, len(rows))
	for i, row := range rows {
		deviations[i] = mapDeviationRow(deviationFields(row))
	}
	result, pageInfo := pagination.BuildPageString(deviations, params.Limit, cursorDir, deviationCreatedAt, deviationID)
	return &domain.ListProductionScheduleDeviationsResult{Deviations: result, PageInfo: pageInfo}, nil
}

func (r *productionScheduleRepoImpl) SumFrozenLines(ctx context.Context, accountID, scheduleID string, frozenThrough time.Time) (*domain.FrozenLineTotals, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.sum_frozen_lines")
	defer span.End()

	row, err := r.queries.SumFrozenProductionScheduleLines(ctx, sqlc.SumFrozenProductionScheduleLinesParams{
		AccountID:            accountID,
		ProductionScheduleID: scheduleID,
		FrozenThroughDate:    frozenThrough,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.FrozenLineTotals{
		LineCount:       row.LineCount,
		PlannedQuantity: decimalToFloat64(row.PlannedQuantity),
	}, nil
}

func (r *productionScheduleRepoImpl) FreezeLines(ctx context.Context, accountID, scheduleID string, frozenThrough time.Time) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.freeze_lines")
	defer span.End()

	err := r.queries.FreezeProductionScheduleLines(ctx, sqlc.FreezeProductionScheduleLinesParams{
		AccountID:            accountID,
		ProductionScheduleID: scheduleID,
		FrozenThroughDate:    frozenThrough,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *productionScheduleRepoImpl) Publish(ctx context.Context, accountID, scheduleID string, frozenThrough time.Time, totals *domain.FrozenLineTotals, publishedByID *string) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.publish")
	defer span.End()

	err := r.queries.PublishProductionSchedule(ctx, sqlc.PublishProductionScheduleParams{
		AccountID:             accountID,
		ID:                    scheduleID,
		FrozenThroughDate:     gosql.NullTime{Time: frozenThrough, Valid: true},
		FrozenLineCount:       safeconv.Int64ToInt32(totals.LineCount),
		FrozenPlannedQuantity: floatToDecimalString(totals.PlannedQuantity),
		PublishedByID:         dtNullString(publishedByID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *productionScheduleRepoImpl) ListPublishedOverlapping(ctx context.Context, accountID, excludeID string, start, end time.Time) ([]string, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.list_published_overlapping")
	defer span.End()

	ids, err := r.queries.ListPublishedProductionSchedulesOverlapping(ctx, sqlc.ListPublishedProductionSchedulesOverlappingParams{
		AccountID:        accountID,
		ExcludeID:        excludeID,
		HorizonStartDate: start,
		HorizonEndDate:   end,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return ids, nil
}

func (r *productionScheduleRepoImpl) Supersede(ctx context.Context, accountID, scheduleID, supersededByID string) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.supersede")
	defer span.End()

	err := r.queries.SupersedeProductionSchedule(ctx, sqlc.SupersedeProductionScheduleParams{
		AccountID:      accountID,
		ID:             scheduleID,
		SupersededByID: gosql.NullString{String: supersededByID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *productionScheduleRepoImpl) SetStatus(ctx context.Context, accountID, scheduleID, statusCode string) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.set_status")
	defer span.End()

	err := r.queries.SetProductionScheduleStatus(ctx, sqlc.SetProductionScheduleStatusParams{
		AccountID:  accountID,
		ID:         scheduleID,
		StatusCode: statusCode,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *productionScheduleRepoImpl) CountReleasedLinesForWeek(ctx context.Context, accountID, scheduleID string, weekIndex int32) (*domain.WeekReleaseState, *apierror.APIError) {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.count_released_lines_for_week")
	defer span.End()

	row, err := r.queries.CountReleasedLinesForWeek(ctx, sqlc.CountReleasedLinesForWeekParams{
		AccountID:            accountID,
		ProductionScheduleID: scheduleID,
		WeekIndex:            weekIndex,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	state := &domain.WeekReleaseState{
		TotalLines:    row.TotalLines,
		ReleasedLines: int64(decimalToFloat64(row.ReleasedLines)),
	}
	// MIN() over a varchar comes back untyped from sqlc, so it needs the same []byte / string coercion any aggregate column does.
	if runID := aggregatedStringOf(row.ExistingProductionRunID); runID != "" {
		state.ExistingProductionRunID = &runID
	}
	return state, nil
}

func (r *productionScheduleRepoImpl) MarkLineReleased(ctx context.Context, accountID, lineID, productionRunID string) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.mark_line_released")
	defer span.End()

	err := r.queries.MarkScheduleLineReleased(ctx, sqlc.MarkScheduleLineReleasedParams{
		AccountID:       accountID,
		ID:              lineID,
		ProductionRunID: gosql.NullString{String: productionRunID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *productionScheduleRepoImpl) UnreleaseLinesForRun(ctx context.Context, accountID, productionRunID string) *apierror.APIError {
	ctx, span := productionScheduleRepoTracer.Start(ctx, "repository.production_schedule.unrelease_lines_for_run")
	defer span.End()

	err := r.queries.UnreleaseScheduleLinesForRun(ctx, sqlc.UnreleaseScheduleLinesForRunParams{
		AccountID:       accountID,
		ProductionRunID: gosql.NullString{String: productionRunID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}
