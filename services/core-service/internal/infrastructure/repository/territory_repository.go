package repository

import (
	"context"
	gosql "database/sql"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var territoryRepoTracer = tracing.GetTracer("core-service.territory_repository")

type territoryRepoImpl struct {
	queries *sqlc.Queries
}

func NewTerritoryRepo(queries *sqlc.Queries) domain.TerritoryRepo {
	return &territoryRepoImpl{queries: queries}
}

func territoryCreatedAt(t *domain.Territory) time.Time { return t.CreatedAt }
func territoryID(t *domain.Territory) string           { return t.ID }

func mapTerritoryRow(
	id string,
	state string,
	startZipcode, endZipcode gosql.NullInt32,
	salesRepID string,
	productLineID gosql.NullString,
	createdAt, updatedAt time.Time,
	salesRepName, salesRepEmail gosql.NullString,
	salesRepStatus string,
	salesRepCreatedAt, salesRepUpdatedAt time.Time,
	productLineName gosql.NullString,
	productLineIsCommissionExempt, productLineIsFreightExempt gosql.NullBool,
	productLineCreatedAt, productLineUpdatedAt gosql.NullTime,
	includes []string,
) *domain.Territory {
	t := &domain.Territory{
		ID:         id,
		State:      state,
		SalesRepID: salesRepID,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}

	if startZipcode.Valid {
		t.StartZipcode = &startZipcode.Int32
	}
	if endZipcode.Valid {
		t.EndZipcode = &endZipcode.Int32
	}

	if slices.Contains(includes, "sales_rep") {
		status := constants.AccountUserStatus(salesRepStatus)
		t.SalesRep = &domain.TerritorySalesRep{
			ID:        salesRepID,
			Status:    &status,
			CreatedAt: &salesRepCreatedAt,
			UpdatedAt: &salesRepUpdatedAt,
		}
		if salesRepName.Valid {
			t.SalesRep.Name = &salesRepName.String
		}
		if salesRepEmail.Valid {
			t.SalesRep.Email = &salesRepEmail.String
		}
	}

	if slices.Contains(includes, "product_line") && productLineID.Valid {
		commPolicy := constants.CommissionPolicyFromBool(productLineIsCommissionExempt.Bool)
		freightPolicy := constants.FreightPolicyFromBool(productLineIsFreightExempt.Bool)
		var plCreatedAt, plUpdatedAt time.Time
		if productLineCreatedAt.Valid {
			plCreatedAt = productLineCreatedAt.Time
		}
		if productLineUpdatedAt.Valid {
			plUpdatedAt = productLineUpdatedAt.Time
		}
		t.ProductLine = &domain.TerritoryProductLine{
			ID:               productLineID.String,
			CommissionPolicy: &commPolicy,
			FreightPolicy:    &freightPolicy,
			CreatedAt:        &plCreatedAt,
			UpdatedAt:        &plUpdatedAt,
		}
		if productLineName.Valid {
			t.ProductLine.Name = productLineName.String
		}
	}

	return t
}

func mapForwardTerritoryRow(row sqlc.ListTerritoriesForwardRow, includes []string) *domain.Territory {
	return mapTerritoryRow(
		row.ID, row.State, row.StartZipcode, row.EndZipcode,
		row.SalesRepID, row.ProductLineID, row.CreatedAt, row.UpdatedAt,
		row.SalesRepName, row.SalesRepEmail,
		row.SalesRepStatus, row.SalesRepCreatedAt, row.SalesRepUpdatedAt,
		row.ProductLineName, row.ProductLineIsCommissionExempt, row.ProductLineIsFreightExempt,
		row.ProductLineCreatedAt, row.ProductLineUpdatedAt,
		includes,
	)
}

func mapBackwardTerritoryRow(row sqlc.ListTerritoriesBackwardRow, includes []string) *domain.Territory {
	return mapTerritoryRow(
		row.ID, row.State, row.StartZipcode, row.EndZipcode,
		row.SalesRepID, row.ProductLineID, row.CreatedAt, row.UpdatedAt,
		row.SalesRepName, row.SalesRepEmail,
		row.SalesRepStatus, row.SalesRepCreatedAt, row.SalesRepUpdatedAt,
		row.ProductLineName, row.ProductLineIsCommissionExempt, row.ProductLineIsFreightExempt,
		row.ProductLineCreatedAt, row.ProductLineUpdatedAt,
		includes,
	)
}

func mapGetTerritoryRow(row sqlc.GetTerritoryRow, includes []string) *domain.Territory {
	return mapTerritoryRow(
		row.ID, row.State, row.StartZipcode, row.EndZipcode,
		row.SalesRepID, row.ProductLineID, row.CreatedAt, row.UpdatedAt,
		row.SalesRepName, row.SalesRepEmail,
		row.SalesRepStatus, row.SalesRepCreatedAt, row.SalesRepUpdatedAt,
		row.ProductLineName, row.ProductLineIsCommissionExempt, row.ProductLineIsFreightExempt,
		row.ProductLineCreatedAt, row.ProductLineUpdatedAt,
		includes,
	)
}

func buildTerritorySearchParams(query *string) (gosql.NullString, gosql.NullInt64) {
	if query == nil || *query == "" {
		return gosql.NullString{}, gosql.NullInt64{}
	}

	searchQuery := gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true}
	zipcodeQuery := parseZipcodeQuery(*query)

	return searchQuery, zipcodeQuery
}

func parseZipcodeQuery(query string) gosql.NullInt64 {
	trimmed := strings.TrimLeft(query, "0")
	if trimmed == "" {
		return gosql.NullInt64{}
	}

	val, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return gosql.NullInt64{}
	}

	if val < 501 || val > 99999 {
		return gosql.NullInt64{}
	}

	return gosql.NullInt64{Int64: val, Valid: true}
}

func (r *territoryRepoImpl) List(ctx context.Context, params domain.ListTerritoriesParams) (*domain.ListTerritoriesResult, *apierror.APIError) {
	ctx, span := territoryRepoTracer.Start(ctx, "repository.territory.list")
	defer span.End()

	searchQuery, zipcodeQuery := buildTerritorySearchParams(params.Query)
	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListTerritoriesBackward(ctx, sqlc.ListTerritoriesBackwardParams{
				AccountID:       params.AccountID,
				SearchQuery:     searchQuery,
				ZipcodeQuery:    zipcodeQuery,
				ZipcodeQuery_2:  zipcodeQuery,
				ZipcodeQuery_3:  zipcodeQuery,
				ZipcodeQuery_4:  zipcodeQuery,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			territories := make([]*domain.Territory, len(rows))
			for i, row := range rows {
				territories[i] = mapBackwardTerritoryRow(row, params.Includes)
			}
			result, pageInfo := pagination.BuildPageString(territories, params.Limit, cursorDir, territoryCreatedAt, territoryID)
			return &domain.ListTerritoriesResult{Territories: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListTerritoriesForward(ctx, sqlc.ListTerritoriesForwardParams{
			AccountID:       params.AccountID,
			SearchQuery:     searchQuery,
			ZipcodeQuery:    zipcodeQuery,
			ZipcodeQuery_2:  zipcodeQuery,
			ZipcodeQuery_3:  zipcodeQuery,
			ZipcodeQuery_4:  zipcodeQuery,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		territories := make([]*domain.Territory, len(rows))
		for i, row := range rows {
			territories[i] = mapForwardTerritoryRow(row, params.Includes)
		}
		result, pageInfo := pagination.BuildPageString(territories, params.Limit, cursorDir, territoryCreatedAt, territoryID)
		return &domain.ListTerritoriesResult{Territories: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListTerritoriesForward(ctx, sqlc.ListTerritoriesForwardParams{
		AccountID:      params.AccountID,
		SearchQuery:    searchQuery,
		ZipcodeQuery:   zipcodeQuery,
		ZipcodeQuery_2: zipcodeQuery,
		ZipcodeQuery_3: zipcodeQuery,
		ZipcodeQuery_4: zipcodeQuery,
		Limit:          params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	territories := make([]*domain.Territory, len(rows))
	for i, row := range rows {
		territories[i] = mapForwardTerritoryRow(row, params.Includes)
	}
	result, pageInfo := pagination.BuildPageString(territories, params.Limit, cursorDir, territoryCreatedAt, territoryID)
	return &domain.ListTerritoriesResult{Territories: result, PageInfo: pageInfo}, nil
}

func (r *territoryRepoImpl) Get(ctx context.Context, params domain.GetTerritoryParams) (*domain.Territory, *apierror.APIError) {
	ctx, span := territoryRepoTracer.Start(ctx, "repository.territory.get")
	defer span.End()

	row, err := r.queries.GetTerritory(ctx, sqlc.GetTerritoryParams{
		ID:        params.TerritoryID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetTerritoryRow(row, params.Includes), nil
}

func (r *territoryRepoImpl) Create(ctx context.Context, territoryID string, params domain.CreateTerritoryParams) (*domain.Territory, *apierror.APIError) {
	ctx, span := territoryRepoTracer.Start(ctx, "repository.territory.create")
	defer span.End()

	// Enforce zipcode null coercion: if start is nil, end must be nil
	endZipcode := params.EndZipcode
	if params.StartZipcode == nil {
		endZipcode = nil
	}

	if err := r.queries.InsertTerritory(ctx, sqlc.InsertTerritoryParams{
		ID:            territoryID,
		State:         params.State,
		StartZipcode:  toNullInt32(params.StartZipcode),
		EndZipcode:    toNullInt32(endZipcode),
		SalesRepID:    params.SalesRepID,
		AccountID:     params.AccountID,
		ProductLineID: toNullString(params.ProductLineID),
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return r.Get(ctx, domain.GetTerritoryParams{
		AccountID:   params.AccountID,
		TerritoryID: territoryID,
		Includes:    params.Includes,
	})
}

func (r *territoryRepoImpl) Update(ctx context.Context, params domain.UpdateTerritoryParams) (*domain.Territory, *apierror.APIError) {
	ctx, span := territoryRepoTracer.Start(ctx, "repository.territory.update")
	defer span.End()

	// Enforce zipcode null coercion: if clearing start, also clear end
	clearEndZipcode := params.ClearEndZipcode
	if params.ClearStartZipcode {
		clearEndZipcode = true
	}

	if err := r.queries.UpdateTerritory(ctx, sqlc.UpdateTerritoryParams{
		ID:                params.TerritoryID,
		AccountID:         params.AccountID,
		State:             toNullString(params.State),
		StartZipcode:      toNullInt32(params.StartZipcode),
		EndZipcode:        toNullInt32(params.EndZipcode),
		SalesRepID:        toNullString(params.SalesRepID),
		ProductLineID:     toNullString(params.ProductLineID),
		ClearProductLine:  params.ClearProductLine,
		ClearStartZipcode: params.ClearStartZipcode,
		ClearEndZipcode:   clearEndZipcode,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return r.Get(ctx, domain.GetTerritoryParams{
		AccountID:   params.AccountID,
		TerritoryID: params.TerritoryID,
		Includes:    params.Includes,
	})
}

func (r *territoryRepoImpl) Delete(ctx context.Context, params domain.DeleteTerritoryParams) *apierror.APIError {
	ctx, span := territoryRepoTracer.Start(ctx, "repository.territory.delete")
	defer span.End()

	if err := r.queries.DeleteTerritory(ctx, sqlc.DeleteTerritoryParams{
		ID:        params.TerritoryID,
		AccountID: params.AccountID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}

	return nil
}

func (r *territoryRepoImpl) IsInAccount(ctx context.Context, accountID, territoryID string) (bool, *apierror.APIError) {
	ctx, span := territoryRepoTracer.Start(ctx, "repository.territory.is_in_account")
	defer span.End()

	exists, err := r.queries.CheckTerritoryInAccount(ctx, sqlc.CheckTerritoryInAccountParams{
		ID:        territoryID,
		AccountID: accountID,
	})
	if err != nil {
		return false, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check territory in account."))
	}
	return exists, nil
}

func (r *territoryRepoImpl) FindSalesRepByZipcode(ctx context.Context, accountID string, zipcode int32) (*string, *apierror.APIError) {
	ctx, span := territoryRepoTracer.Start(ctx, "repository.territory.find_sales_rep_by_zipcode")
	defer span.End()

	salesRepID, err := r.queries.FindSalesRepByZipcode(ctx, sqlc.FindSalesRepByZipcodeParams{
		AccountID: accountID,
		Zipcode:   gosql.NullInt32{Int32: zipcode, Valid: true},
	})
	if err != nil {
		if apierror.IsNotFound(db.MapSQLError(err)) {
			return nil, nil
		}
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to look up sales rep by zipcode."))
	}
	return &salesRepID, nil
}

func (r *territoryRepoImpl) FindSalesRepByState(ctx context.Context, accountID, state string) (*string, *apierror.APIError) {
	ctx, span := territoryRepoTracer.Start(ctx, "repository.territory.find_sales_rep_by_state")
	defer span.End()

	salesRepID, err := r.queries.FindSalesRepByState(ctx, sqlc.FindSalesRepByStateParams{
		AccountID: accountID,
		State:     state,
	})
	if err != nil {
		if apierror.IsNotFound(db.MapSQLError(err)) {
			return nil, nil
		}
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to look up sales rep by state."))
	}
	return &salesRepID, nil
}
