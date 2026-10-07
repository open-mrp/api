package repository

import (
	"context"
	gosql "database/sql"
	"slices"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
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
	if productLineID.Valid {
		t.ProductLineID = &productLineID.String
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

func (r *territoryRepoImpl) List(ctx context.Context, params domain.ListTerritoriesParams) (*domain.ListTerritoriesResult, *apierror.APIError) {
	ctx, span := territoryRepoTracer.Start(ctx, "repository.territory.list")
	defer span.End()

	cursor, apiErr := decodeListCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	query, args := territoryListPageQuery(params.AccountID, params.Query, cursor, params.Limit+1)
	ids, err := selectStrings(ctx, r.queries.DB(), query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	byID := make(map[string]*domain.Territory, len(ids))
	if len(ids) > 0 {
		rows, err := r.queries.GetTerritoriesByIDs(ctx, sqlc.GetTerritoriesByIDsParams{
			Ids:       ids,
			AccountID: params.AccountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, row := range rows {
			byID[row.ID] = mapGetTerritoriesByIDsRow(row, params.Includes)
		}
	}

	result, pageInfo := pagination.BuildPageString(inPageOrder(ids, byID), params.Limit, cursorDirection(cursor), territoryCreatedAt, territoryID)
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

	if err := r.queries.InsertTerritory(ctx, sqlc.InsertTerritoryParams{
		ID:            territoryID,
		State:         params.State,
		StartZipcode:  toNullInt32(params.StartZipcode),
		EndZipcode:    toNullInt32(params.EndZipcode),
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
		ClearEndZipcode:   params.ClearEndZipcode,
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

func mapGetTerritoriesByIDsRow(row sqlc.GetTerritoriesByIDsRow, includes []string) *domain.Territory {
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

func (r *territoryRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.Territory, *apierror.APIError) {
	ctx, span := territoryRepoTracer.Start(ctx, "repository.territory.get_by_ids")
	defer span.End()

	rows, err := r.queries.GetTerritoriesByIDs(ctx, sqlc.GetTerritoriesByIDsParams{
		Ids:       ids,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	territories := make([]*domain.Territory, len(rows))
	for i, row := range rows {
		territories[i] = mapGetTerritoriesByIDsRow(row, []string{"sales_rep", "product_line"})
	}
	return territories, nil
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
