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
	"github.com/augno/api/shared/tracing"
)

var demandOverrideRepoTracer = tracing.GetTracer("core-service.demand_override_repository")

type demandOverrideRepoImpl struct {
	queries *sqlc.Queries
}

func NewDemandOverrideRepo(queries *sqlc.Queries) domain.DemandOverrideRepo {
	return &demandOverrideRepoImpl{queries: queries}
}

func demandOverrideCreatedAt(o *domain.DemandOverride) time.Time { return o.CreatedAt }
func demandOverrideID(o *domain.DemandOverride) string           { return o.ID }

func doNullFloatString(v *float64) gosql.NullString {
	if v == nil {
		return gosql.NullString{}
	}
	return gosql.NullString{String: floatToDecimalString(*v), Valid: true}
}

func doNullBool(b *bool) gosql.NullBool {
	if b == nil {
		return gosql.NullBool{}
	}
	return gosql.NullBool{Bool: *b, Valid: true}
}

// demandOverrideFields is the shared column set every override query selects. sqlc emits a distinct row struct per query with an identical shape, so each mapper converts into this one type rather than duplicating the nil-handling.
type demandOverrideFields struct {
	ID               string
	AccountID        string
	ScopeCode        string
	ScopeRefID       string
	PeriodStartDate  time.Time
	PeriodEndDate    time.Time
	OverrideTypeCode string
	Value            string
	UnitID           gosql.NullString
	ReasonCode       gosql.NullString
	Note             gosql.NullString
	CreatedByID      string
	EffectiveFrom    time.Time
	ExpiresAt        gosql.NullTime
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ScopeName        string
	ScopeHandle      gosql.NullString
}

func mapDemandOverride(f demandOverrideFields) *domain.DemandOverride {
	o := &domain.DemandOverride{
		ID:               f.ID,
		AccountID:        f.AccountID,
		ScopeCode:        f.ScopeCode,
		ScopeRefID:       f.ScopeRefID,
		PeriodStartDate:  f.PeriodStartDate,
		PeriodEndDate:    f.PeriodEndDate,
		OverrideTypeCode: f.OverrideTypeCode,
		Value:            decimalToFloat64(f.Value),
		CreatedByID:      f.CreatedByID,
		EffectiveFrom:    f.EffectiveFrom,
		IsActive:         f.IsActive,
		CreatedAt:        f.CreatedAt,
		UpdatedAt:        f.UpdatedAt,
	}
	if f.UnitID.Valid {
		o.UnitID = &f.UnitID.String
	}
	if f.ReasonCode.Valid {
		o.ReasonCode = &f.ReasonCode.String
	}
	if f.Note.Valid {
		o.Note = &f.Note.String
	}
	if f.ExpiresAt.Valid {
		o.ExpiresAt = &f.ExpiresAt.Time
	}
	if f.ScopeName != "" {
		o.ScopeName = &f.ScopeName
	}
	if f.ScopeHandle.Valid {
		o.ScopeHandle = &f.ScopeHandle.String
	}
	return o
}

func (r *demandOverrideRepoImpl) ListTypes(ctx context.Context) ([]*domain.DemandOverrideType, *apierror.APIError) {
	ctx, span := demandOverrideRepoTracer.Start(ctx, "repository.demand_override.list_types")
	defer span.End()

	rows, err := r.queries.ListDemandOverrideTypes(ctx)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	types := make([]*domain.DemandOverrideType, len(rows))
	for i, row := range rows {
		types[i] = &domain.DemandOverrideType{
			ID:        row.ID,
			Code:      row.Code,
			Name:      row.Name,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
	}
	return types, nil
}

func (r *demandOverrideRepoImpl) List(ctx context.Context, params domain.ListDemandOverridesParams) (*domain.ListDemandOverridesResult, *apierror.APIError) {
	ctx, span := demandOverrideRepoTracer.Start(ctx, "repository.demand_override.list")
	defer span.End()

	searchQuery := dtSearchParam(params.Query)
	includeScopeFilter := len(params.ScopeCodes) > 0
	includeScopeRefFilter := len(params.ScopeRefIDs) > 0
	includeTypeFilter := len(params.OverrideTypeCodes) > 0
	isActive := doNullBool(params.IsActive)
	periodStart := dtNullTime(params.PeriodStart)
	periodEnd := dtNullTime(params.PeriodEnd)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListDemandOverridesBackward(ctx, sqlc.ListDemandOverridesBackwardParams{
				AccountID:             params.AccountID,
				IncludeScopeFilter:    includeScopeFilter,
				ScopeCodes:            params.ScopeCodes,
				IncludeScopeRefFilter: includeScopeRefFilter,
				ScopeRefIds:           params.ScopeRefIDs,
				IncludeTypeFilter:     includeTypeFilter,
				OverrideTypeCodes:     params.OverrideTypeCodes,
				IsActive:              isActive,
				PeriodStart:           periodStart,
				PeriodEnd:             periodEnd,
				SearchQuery:           searchQuery,
				CursorCreatedAt:       cur.OccurredAt,
				CursorID:              cur.ID,
				Limit:                 params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			overrides := make([]*domain.DemandOverride, len(rows))
			for i, row := range rows {
				overrides[i] = mapDemandOverride(demandOverrideFields(row))
			}
			result, pageInfo := pagination.BuildPageString(overrides, params.Limit, cursorDir, demandOverrideCreatedAt, demandOverrideID)
			return &domain.ListDemandOverridesResult{Overrides: result, PageInfo: pageInfo}, nil
		}

		rows, err := r.queries.ListDemandOverridesForward(ctx, sqlc.ListDemandOverridesForwardParams{
			AccountID:             params.AccountID,
			IncludeScopeFilter:    includeScopeFilter,
			ScopeCodes:            params.ScopeCodes,
			IncludeScopeRefFilter: includeScopeRefFilter,
			ScopeRefIds:           params.ScopeRefIDs,
			IncludeTypeFilter:     includeTypeFilter,
			OverrideTypeCodes:     params.OverrideTypeCodes,
			IsActive:              isActive,
			PeriodStart:           periodStart,
			PeriodEnd:             periodEnd,
			SearchQuery:           searchQuery,
			CursorCreatedAt:       gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:              gosql.NullString{String: cur.ID, Valid: true},
			Limit:                 params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		overrides := make([]*domain.DemandOverride, len(rows))
		for i, row := range rows {
			overrides[i] = mapDemandOverride(demandOverrideFields(row))
		}
		result, pageInfo := pagination.BuildPageString(overrides, params.Limit, cursorDir, demandOverrideCreatedAt, demandOverrideID)
		return &domain.ListDemandOverridesResult{Overrides: result, PageInfo: pageInfo}, nil
	}

	rows, err := r.queries.ListDemandOverridesForward(ctx, sqlc.ListDemandOverridesForwardParams{
		AccountID:             params.AccountID,
		IncludeScopeFilter:    includeScopeFilter,
		ScopeCodes:            params.ScopeCodes,
		IncludeScopeRefFilter: includeScopeRefFilter,
		ScopeRefIds:           params.ScopeRefIDs,
		IncludeTypeFilter:     includeTypeFilter,
		OverrideTypeCodes:     params.OverrideTypeCodes,
		IsActive:              isActive,
		PeriodStart:           periodStart,
		PeriodEnd:             periodEnd,
		SearchQuery:           searchQuery,
		Limit:                 params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	overrides := make([]*domain.DemandOverride, len(rows))
	for i, row := range rows {
		overrides[i] = mapDemandOverride(demandOverrideFields(row))
	}
	result, pageInfo := pagination.BuildPageString(overrides, params.Limit, cursorDir, demandOverrideCreatedAt, demandOverrideID)
	return &domain.ListDemandOverridesResult{Overrides: result, PageInfo: pageInfo}, nil
}

func (r *demandOverrideRepoImpl) Get(ctx context.Context, params domain.GetDemandOverrideParams) (*domain.DemandOverride, *apierror.APIError) {
	ctx, span := demandOverrideRepoTracer.Start(ctx, "repository.demand_override.get")
	defer span.End()

	row, err := r.queries.GetDemandOverride(ctx, sqlc.GetDemandOverrideParams{
		AccountID: params.AccountID,
		ID:        params.OverrideID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return mapDemandOverride(demandOverrideFields(row)), nil
}

func (r *demandOverrideRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.DemandOverride, *apierror.APIError) {
	ctx, span := demandOverrideRepoTracer.Start(ctx, "repository.demand_override.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := r.queries.GetDemandOverridesByIDs(ctx, sqlc.GetDemandOverridesByIDsParams{
		AccountID: accountID,
		Ids:       ids,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	overrides := make([]*domain.DemandOverride, len(rows))
	for i, row := range rows {
		overrides[i] = mapDemandOverride(demandOverrideFields(row))
	}
	return overrides, nil
}

func (r *demandOverrideRepoImpl) Create(ctx context.Context, id string, override *domain.DemandOverride) (*domain.DemandOverride, *apierror.APIError) {
	ctx, span := demandOverrideRepoTracer.Start(ctx, "repository.demand_override.create")
	defer span.End()

	err := r.queries.CreateDemandOverride(ctx, sqlc.CreateDemandOverrideParams{
		ID:               id,
		AccountID:        override.AccountID,
		ScopeCode:        override.ScopeCode,
		ScopeRefID:       override.ScopeRefID,
		PeriodStartDate:  override.PeriodStartDate,
		PeriodEndDate:    override.PeriodEndDate,
		OverrideTypeCode: override.OverrideTypeCode,
		Value:            floatToDecimalString(override.Value),
		UnitID:           dtNullString(override.UnitID),
		ReasonCode:       dtNullString(override.ReasonCode),
		Note:             dtNullString(override.Note),
		CreatedByID:      override.CreatedByID,
		EffectiveFrom:    override.EffectiveFrom,
		ExpiresAt:        dtNullTime(override.ExpiresAt),
		IsActive:         override.IsActive,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetDemandOverrideParams{AccountID: override.AccountID, OverrideID: id})
}

func (r *demandOverrideRepoImpl) Update(ctx context.Context, params domain.UpdateDemandOverrideParams) (*domain.DemandOverride, *apierror.APIError) {
	ctx, span := demandOverrideRepoTracer.Start(ctx, "repository.demand_override.update")
	defer span.End()

	err := r.queries.UpdateDemandOverride(ctx, sqlc.UpdateDemandOverrideParams{
		AccountID:        params.AccountID,
		ID:               params.OverrideID,
		PeriodStartDate:  dtNullTime(params.PeriodStartDate),
		PeriodEndDate:    dtNullTime(params.PeriodEndDate),
		OverrideTypeCode: dtNullString(params.OverrideTypeCode),
		Value:            doNullFloatString(params.Value),
		UnitID:           dtNullString(params.UnitID.ValuePtr()),
		ClearUnitID:      params.UnitID.IsClear(),
		ReasonCode:       dtNullString(params.ReasonCode.ValuePtr()),
		ClearReasonCode:  params.ReasonCode.IsClear(),
		Note:             dtNullString(params.Note.ValuePtr()),
		ClearNote:        params.Note.IsClear(),
		ExpiresAt:        dtNullTime(params.ExpiresAt.ValuePtr()),
		ClearExpiresAt:   params.ExpiresAt.IsClear(),
		IsActive:         doNullBool(params.IsActive),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetDemandOverrideParams{AccountID: params.AccountID, OverrideID: params.OverrideID})
}

func (r *demandOverrideRepoImpl) Delete(ctx context.Context, params domain.DeleteDemandOverrideParams) *apierror.APIError {
	ctx, span := demandOverrideRepoTracer.Start(ctx, "repository.demand_override.delete")
	defer span.End()

	err := r.queries.DeleteDemandOverride(ctx, sqlc.DeleteDemandOverrideParams{
		AccountID: params.AccountID,
		ID:        params.OverrideID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}
